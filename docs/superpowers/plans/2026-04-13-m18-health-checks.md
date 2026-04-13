# M18 — Health Checks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a health check framework that runs checks on startup and periodically, exposes results via `/health` API endpoints, and dispatches notifications for new health issues.

**Architecture:** A `health.Check` interface with a `Checker` runner that aggregates results from individual checks. Each check is a separate file in `internal/health/`. The `Checker` is wired into the API Deps for serving via v3/v6 endpoints, and a scheduled task runs checks every 30 minutes with notification dispatch for new issues.

**Tech Stack:** Go stdlib only — no new dependencies

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `internal/health/health.go` | Types (Level, Result, Check interface) and Checker runner |
| Create | `internal/health/health_test.go` | Checker unit tests |
| Create | `internal/health/database.go` | DatabaseCheck — pings DB |
| Create | `internal/health/database_test.go` | DatabaseCheck tests |
| Create | `internal/health/rootfolder.go` | RootFolderCheck — verifies series root folders exist |
| Create | `internal/health/rootfolder_test.go` | RootFolderCheck tests |
| Create | `internal/health/indexer.go` | IndexerCheck — at least one indexer enabled |
| Create | `internal/health/indexer_test.go` | IndexerCheck tests |
| Create | `internal/health/downloadclient.go` | DownloadClientCheck — at least one DC enabled |
| Create | `internal/health/downloadclient_test.go` | DownloadClientCheck tests |
| Create | `internal/health/metadata.go` | MetadataSourceCheck — TVDB API key configured |
| Create | `internal/health/metadata_test.go` | MetadataSourceCheck tests |
| Modify | `internal/api/server.go` | Add HealthChecker to Deps |
| Modify | `internal/api/v3/health.go` | Replace stub with real results from Checker |
| Modify | `internal/api/v3/task6_test.go` | Update health test for new behavior |
| Create | `internal/api/v6/health.go` | V6 health endpoint |
| Modify | `internal/api/v6/v6.go` | Mount v6 health route |
| Modify | `internal/app/app.go` | Wire Checker, add HealthCheck handler + scheduled task, notification dispatch |

---

### Task 1: Health Check Types and Checker

**Files:**
- Create: `internal/health/health.go`
- Create: `internal/health/health_test.go`

- [ ] **Step 1: Write the failing test for Checker**

Create `internal/health/health_test.go`:

```go
package health

import (
	"context"
	"testing"
)

// fakeCheck returns fixed results.
type fakeCheck struct {
	name    string
	results []Result
}

func (f *fakeCheck) Name() string                        { return f.name }
func (f *fakeCheck) Check(_ context.Context) []Result    { return f.results }

func TestCheckerRunAll(t *testing.T) {
	c := NewChecker(
		&fakeCheck{name: "OK", results: nil},
		&fakeCheck{name: "Warn", results: []Result{
			{Source: "Warn", Type: LevelWarning, Message: "something wrong"},
		}},
		&fakeCheck{name: "Multi", results: []Result{
			{Source: "Multi", Type: LevelError, Message: "error 1"},
			{Source: "Multi", Type: LevelError, Message: "error 2"},
		}},
	)

	results := c.RunAll(context.Background())
	if len(results) != 3 {
		t.Fatalf("RunAll returned %d results, want 3", len(results))
	}

	// Verify results are cached.
	cached := c.Results()
	if len(cached) != 3 {
		t.Fatalf("Results() returned %d results, want 3", len(cached))
	}
}

func TestCheckerEmpty(t *testing.T) {
	c := NewChecker()
	results := c.RunAll(context.Background())
	if len(results) != 0 {
		t.Fatalf("empty checker returned %d results, want 0", len(results))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/health/ -run TestChecker -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write the health package**

Create `internal/health/health.go`:

```go
// Package health provides a framework for running health checks and
// aggregating results. Individual checks implement the Check interface.
package health

import (
	"context"
	"sync"
)

// Level indicates the severity of a health check result.
type Level string

const (
	LevelOK      Level = "ok"
	LevelNotice  Level = "notice"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

// Result is a single health check finding.
type Result struct {
	Source  string `json:"source"`
	Type    Level  `json:"type"`
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl,omitempty"`
}

// Check is a single health check that evaluates one aspect of system health.
type Check interface {
	Name() string
	Check(ctx context.Context) []Result
}

// Checker runs all registered health checks and aggregates results.
type Checker struct {
	checks []Check
	mu     sync.RWMutex
	last   []Result
}

// NewChecker creates a Checker with the given checks.
func NewChecker(checks ...Check) *Checker {
	return &Checker{checks: checks}
}

// RunAll runs every check and caches the aggregated results.
func (c *Checker) RunAll(ctx context.Context) []Result {
	var all []Result
	for _, ch := range c.checks {
		all = append(all, ch.Check(ctx)...)
	}
	if all == nil {
		all = []Result{}
	}
	c.mu.Lock()
	c.last = all
	c.mu.Unlock()
	return all
}

// Results returns the most recently cached results.
func (c *Checker) Results() []Result {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.last == nil {
		return []Result{}
	}
	return c.last
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/health/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/health/health.go internal/health/health_test.go
git commit -m "feat(health): add health check framework with Check interface and Checker runner"
```

---

### Task 2: Individual Health Checks

**Files:**
- Create: `internal/health/database.go`
- Create: `internal/health/database_test.go`
- Create: `internal/health/rootfolder.go`
- Create: `internal/health/rootfolder_test.go`
- Create: `internal/health/indexer.go`
- Create: `internal/health/indexer_test.go`
- Create: `internal/health/downloadclient.go`
- Create: `internal/health/downloadclient_test.go`
- Create: `internal/health/metadata.go`
- Create: `internal/health/metadata_test.go`

- [ ] **Step 1: Write failing test for DatabaseCheck**

Create `internal/health/database_test.go`:

```go
package health

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct {
	err error
}

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

func TestDatabaseCheckHealthy(t *testing.T) {
	c := NewDatabaseCheck(&fakePinger{})
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("healthy DB returned %d results, want 0", len(results))
	}
}

func TestDatabaseCheckUnhealthy(t *testing.T) {
	c := NewDatabaseCheck(&fakePinger{err: errors.New("connection refused")})
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("unhealthy DB returned %d results, want 1", len(results))
	}
	if results[0].Type != LevelError {
		t.Errorf("Type = %q, want error", results[0].Type)
	}
	if results[0].Source != "DatabaseCheck" {
		t.Errorf("Source = %q, want DatabaseCheck", results[0].Source)
	}
}
```

- [ ] **Step 2: Implement DatabaseCheck**

Create `internal/health/database.go`:

```go
package health

import (
	"context"
	"fmt"
)

// Pinger is the subset of db.Pool needed for health checking.
type Pinger interface {
	Ping(ctx context.Context) error
}

// DatabaseCheck verifies the database is reachable.
type DatabaseCheck struct {
	pool Pinger
}

func NewDatabaseCheck(pool Pinger) *DatabaseCheck {
	return &DatabaseCheck{pool: pool}
}

func (c *DatabaseCheck) Name() string { return "DatabaseCheck" }

func (c *DatabaseCheck) Check(ctx context.Context) []Result {
	if err := c.pool.Ping(ctx); err != nil {
		return []Result{{
			Source:  "DatabaseCheck",
			Type:    LevelError,
			Message: fmt.Sprintf("Unable to connect to database: %v", err),
		}}
	}
	return nil
}
```

- [ ] **Step 3: Write failing tests for remaining checks**

Create `internal/health/rootfolder_test.go`:

```go
package health

import (
	"context"
	"testing"
)

type fakeSeriesLister struct {
	paths []string
}

func (f *fakeSeriesLister) ListRootPaths(_ context.Context) ([]string, error) {
	return f.paths, nil
}

func TestRootFolderCheckAllExist(t *testing.T) {
	c := NewRootFolderCheck(&fakeSeriesLister{paths: []string{t.TempDir()}})
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("all folders exist, got %d results", len(results))
	}
}

func TestRootFolderCheckMissing(t *testing.T) {
	c := NewRootFolderCheck(&fakeSeriesLister{paths: []string{"/nonexistent/path/xyz"}})
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("missing folder returned %d results, want 1", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("Type = %q, want warning", results[0].Type)
	}
}
```

Create `internal/health/indexer_test.go`:

```go
package health

import (
	"context"
	"testing"
)

type fakeInstanceCounter struct {
	count int
}

func (f *fakeInstanceCounter) CountEnabled(_ context.Context) (int, error) {
	return f.count, nil
}

func TestIndexerCheckHasIndexer(t *testing.T) {
	c := NewIndexerCheck(&fakeInstanceCounter{count: 1})
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("has indexer, got %d results", len(results))
	}
}

func TestIndexerCheckNoIndexer(t *testing.T) {
	c := NewIndexerCheck(&fakeInstanceCounter{count: 0})
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("no indexer, got %d results, want 1", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("Type = %q, want warning", results[0].Type)
	}
}
```

Create `internal/health/downloadclient_test.go`:

```go
package health

import (
	"context"
	"testing"
)

func TestDownloadClientCheckHasClient(t *testing.T) {
	c := NewDownloadClientCheck(&fakeInstanceCounter{count: 2})
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("has client, got %d results", len(results))
	}
}

func TestDownloadClientCheckNoClient(t *testing.T) {
	c := NewDownloadClientCheck(&fakeInstanceCounter{count: 0})
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("no client, got %d results, want 1", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("Type = %q, want warning", results[0].Type)
	}
}
```

Create `internal/health/metadata_test.go`:

```go
package health

import (
	"context"
	"testing"
)

func TestMetadataSourceCheckConfigured(t *testing.T) {
	c := NewMetadataSourceCheck("my-api-key")
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("configured key, got %d results", len(results))
	}
}

func TestMetadataSourceCheckUnconfigured(t *testing.T) {
	c := NewMetadataSourceCheck("")
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("empty key, got %d results, want 1", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("Type = %q, want warning", results[0].Type)
	}
}
```

- [ ] **Step 4: Implement all remaining checks**

Create `internal/health/rootfolder.go`:

```go
package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// SeriesPathLister lists distinct root folder paths derived from series.
type SeriesPathLister interface {
	ListRootPaths(ctx context.Context) ([]string, error)
}

// RootFolderCheck verifies that series root folders exist on disk.
type RootFolderCheck struct {
	lister SeriesPathLister
}

func NewRootFolderCheck(lister SeriesPathLister) *RootFolderCheck {
	return &RootFolderCheck{lister: lister}
}

func (c *RootFolderCheck) Name() string { return "RootFolderCheck" }

func (c *RootFolderCheck) Check(ctx context.Context) []Result {
	paths, err := c.lister.ListRootPaths(ctx)
	if err != nil {
		return []Result{{
			Source:  "RootFolderCheck",
			Type:    LevelError,
			Message: fmt.Sprintf("Failed to list root paths: %v", err),
		}}
	}

	var results []Result
	seen := map[string]bool{}
	for _, p := range paths {
		root := filepath.Dir(p)
		if seen[root] {
			continue
		}
		seen[root] = true
		if _, err := os.Stat(root); os.IsNotExist(err) {
			results = append(results, Result{
				Source:  "RootFolderCheck",
				Type:    LevelWarning,
				Message: fmt.Sprintf("Root folder is missing: %s", root),
			})
		}
	}
	return results
}
```

Create `internal/health/indexer.go`:

```go
package health

import "context"

// EnabledCounter counts the number of enabled provider instances.
type EnabledCounter interface {
	CountEnabled(ctx context.Context) (int, error)
}

// IndexerCheck warns if no indexers are configured.
type IndexerCheck struct {
	counter EnabledCounter
}

func NewIndexerCheck(counter EnabledCounter) *IndexerCheck {
	return &IndexerCheck{counter: counter}
}

func (c *IndexerCheck) Name() string { return "IndexerCheck" }

func (c *IndexerCheck) Check(ctx context.Context) []Result {
	n, err := c.counter.CountEnabled(ctx)
	if err != nil {
		return []Result{{
			Source:  "IndexerCheck",
			Type:    LevelError,
			Message: "Failed to check indexer configuration",
		}}
	}
	if n == 0 {
		return []Result{{
			Source:  "IndexerCheck",
			Type:    LevelWarning,
			Message: "No indexers are enabled. Sonarr will not be able to find new releases automatically",
		}}
	}
	return nil
}
```

Create `internal/health/downloadclient.go`:

```go
package health

import "context"

// DownloadClientCheck warns if no download clients are configured.
type DownloadClientCheck struct {
	counter EnabledCounter
}

func NewDownloadClientCheck(counter EnabledCounter) *DownloadClientCheck {
	return &DownloadClientCheck{counter: counter}
}

func (c *DownloadClientCheck) Name() string { return "DownloadClientCheck" }

func (c *DownloadClientCheck) Check(ctx context.Context) []Result {
	n, err := c.counter.CountEnabled(ctx)
	if err != nil {
		return []Result{{
			Source:  "DownloadClientCheck",
			Type:    LevelError,
			Message: "Failed to check download client configuration",
		}}
	}
	if n == 0 {
		return []Result{{
			Source:  "DownloadClientCheck",
			Type:    LevelWarning,
			Message: "No download clients are enabled. Sonarr will not be able to download releases",
		}}
	}
	return nil
}
```

Create `internal/health/metadata.go`:

```go
package health

import "context"

// MetadataSourceCheck warns if the TVDB API key is not configured.
type MetadataSourceCheck struct {
	apiKey string
}

func NewMetadataSourceCheck(apiKey string) *MetadataSourceCheck {
	return &MetadataSourceCheck{apiKey: apiKey}
}

func (c *MetadataSourceCheck) Name() string { return "MetadataSourceCheck" }

func (c *MetadataSourceCheck) Check(_ context.Context) []Result {
	if c.apiKey == "" {
		return []Result{{
			Source:  "MetadataSourceCheck",
			Type:    LevelWarning,
			Message: "TVDB API key is not configured. Series metadata refresh will not work",
		}}
	}
	return nil
}
```

- [ ] **Step 5: Run all health tests**

Run: `go test ./internal/health/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/health/
git commit -m "feat(health): add database, root folder, indexer, download client, and metadata checks"
```

---

### Task 3: CountEnabled Methods for Instance Stores

**Files:**
- Modify: `internal/providers/indexer/store.go` (or the file containing InstanceStore)
- Modify: `internal/providers/downloadclient/store.go`

The health checks need `CountEnabled(ctx) (int, error)` on the instance stores. Currently the stores have `List(ctx)` which returns all instances. We need to add a method that counts only enabled ones, or the health check can use `List` and count locally.

- [ ] **Step 1: Check existing InstanceStore interfaces**

Read the indexer and download client InstanceStore interfaces to see what methods exist.

- [ ] **Step 2: Implement CountEnabled or use List-and-count adapter**

If the store doesn't have `CountEnabled`, create a simple adapter in the health package that wraps `List` and counts entries with `Enable: true`:

In `internal/health/indexer.go`, if needed, add an adapter:

```go
// InstanceLister is the subset of indexer.InstanceStore needed.
type InstanceLister interface {
	List(ctx context.Context) ([]Instance, error)
}
```

However, since the health checks use a minimal interface (`EnabledCounter`), the simplest approach is to implement `CountEnabled` in the health check constructors by accepting a function or by wrapping `List`. Decide based on what the store interfaces look like.

The key constraint: do NOT import the indexer or downloadclient packages from the health package. Use interfaces.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/health/ -v`

- [ ] **Step 4: Commit if changes were needed**

---

### Task 4: API Endpoints

**Files:**
- Modify: `internal/api/server.go` — add `HealthChecker` to `Deps`
- Modify: `internal/api/v3/health.go` — replace stub
- Modify: `internal/api/v3/task6_test.go` — update test
- Create: `internal/api/v6/health.go` — v6 health endpoint
- Modify: `internal/api/v6/v6.go` — mount health route

- [ ] **Step 1: Add HealthChecker to Deps**

In `internal/api/server.go`, add to the `Deps` struct:

```go
HealthChecker HealthResultsProvider
```

Add a minimal interface (avoids importing the health package from api):

```go
// HealthResultsProvider returns cached health check results.
type HealthResultsProvider interface {
	Results() []HealthResult
}

// HealthResult is a single health check finding (mirrors health.Result).
type HealthResult struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl,omitempty"`
}
```

Wait — this duplicates the type. Better approach: define the result type in the api package as a JSON struct, and have the v3/v6 handlers call a function that converts. Or, simpler: just use `any` and let the health.Checker return JSON-serializable results directly.

Simplest approach that avoids circular imports: the `Deps` struct holds an `interface{ Results() []any }` or a concrete `*health.Checker`. Since `api` doesn't import `health` currently, we can either:
a) Import `health` from `api` (it's a leaf package, no cycles)
b) Use an interface

Option (a) is cleanest — `health` has no imports from `api`.

Add to `internal/api/server.go` imports:

```go
"github.com/ajthom90/sonarr2/internal/health"
```

Add to `Deps`:

```go
HealthChecker *health.Checker
```

- [ ] **Step 2: Replace v3 health stub**

Replace `internal/api/v3/health.go`:

```go
package v3

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/go-chi/chi/v5"
)

// MountHealth registers /api/v3/health routes.
func MountHealth(r chi.Router, checker *health.Checker) {
	r.Route("/api/v3/health", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			results := checker.Results()
			writeJSON(w, http.StatusOK, results)
		})
	})
}
```

Update the call site in `server.go` where `MountHealth` is called — pass `deps.HealthChecker`.

- [ ] **Step 3: Create v6 health endpoint**

Create `internal/api/v6/health.go`:

```go
package v6

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/health"
)

func (d *Deps) mountHealth() {
	d.router.Get("/api/v6/health", func(w http.ResponseWriter, r *http.Request) {
		results := d.HealthChecker.Results()
		writeJSON(w, http.StatusOK, results)
	})
}
```

Add `HealthChecker *health.Checker` to the v6 `Deps` struct and call `d.mountHealth()` in the v6 mount function.

- [ ] **Step 4: Update tests**

Update `internal/api/v3/task6_test.go` to pass a `*health.Checker` to `MountHealth`.

- [ ] **Step 5: Build and test**

Run: `go build ./... && go test ./internal/api/... -v`

- [ ] **Step 6: Commit**

```bash
git add internal/api/ internal/health/
git commit -m "feat(api): wire health check results into v3 and v6 endpoints"
```

---

### Task 5: Wire in app.go + Scheduled Task + Notification Dispatch

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add health package import**

Add to imports:

```go
"github.com/ajthom90/sonarr2/internal/health"
```

- [ ] **Step 2: Create the Checker after all stores are wired**

After the engine/grab/rss section, before building the server:

```go
// Health checks.
checker := health.NewChecker(
	health.NewDatabaseCheck(pool),
	health.NewRootFolderCheck(&rootPathAdapter{series: lib.Series}),
	health.NewIndexerCheck(&enabledCountAdapter{store: idxStore}),
	health.NewDownloadClientCheck(&enabledCountAdapter{store: dcStore}),
	health.NewMetadataSourceCheck(cfg.TVDB.ApiKey),
)

// Run initial health check.
checker.RunAll(ctx)
```

Add the `Checker` to the API `Deps` struct:

```go
HealthChecker: checker,
```

- [ ] **Step 3: Add adapter types for health check interfaces**

Add small adapter types at the bottom of app.go:

```go
// rootPathAdapter adapts library.SeriesStore for health.SeriesPathLister.
type rootPathAdapter struct {
	series library.SeriesStore
}

func (a *rootPathAdapter) ListRootPaths(ctx context.Context) ([]string, error) {
	all, err := a.series.List(ctx)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(all))
	for i, s := range all {
		paths[i] = s.Path
	}
	return paths, nil
}

// enabledCountAdapter adapts an InstanceStore for health.EnabledCounter.
type enabledCountAdapter struct {
	store interface {
		List(ctx context.Context) ([]interface{ IsEnabled() bool }, error)
	}
}
```

Actually, since indexer.InstanceStore and downloadclient.InstanceStore return different types, use two separate adapters or a generic approach. The simplest is:

```go
type indexerCountAdapter struct {
	store indexer.InstanceStore
}

func (a *indexerCountAdapter) CountEnabled(ctx context.Context) (int, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, inst := range all {
		if inst.Enable {
			n++
		}
	}
	return n, nil
}

type dcCountAdapter struct {
	store downloadclient.InstanceStore
}

func (a *dcCountAdapter) CountEnabled(ctx context.Context) (int, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, inst := range all {
		if inst.Enable {
			n++
		}
	}
	return n, nil
}
```

- [ ] **Step 4: Add HealthCheck command handler and scheduled task**

Register a HealthCheck handler that runs the checker and dispatches notifications:

```go
healthHandler := &healthCheckHandler{
	checker:    checker,
	notifStore: notifStore,
	notifReg:   notifReg,
	log:        log,
}
reg.Register("HealthCheck", healthHandler)

// Schedule HealthCheck at 30-minute interval.
if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
	TypeName:      "HealthCheck",
	IntervalSecs:  1800,
	NextExecution: time.Now().Add(30 * time.Minute),
}); err != nil {
	_ = pool.Close()
	return nil, fmt.Errorf("app: upsert HealthCheck task: %w", err)
}
```

Add the handler type:

```go
// healthCheckHandler runs health checks and dispatches notifications for new issues.
type healthCheckHandler struct {
	checker    *health.Checker
	notifStore notification.InstanceStore
	notifReg   *notification.Registry
	log        *slog.Logger
	lastIssues map[string]bool // tracks which issues were reported last run
}

func (h *healthCheckHandler) Handle(ctx context.Context, _ commands.Command) error {
	results := h.checker.RunAll(ctx)

	// Find new issues (not in last run).
	currentIssues := map[string]bool{}
	for _, r := range results {
		if r.Type == health.LevelWarning || r.Type == health.LevelError {
			key := r.Source + ":" + r.Message
			currentIssues[key] = true
			if h.lastIssues != nil && h.lastIssues[key] {
				continue // already reported
			}
			// New issue — dispatch notifications.
			dispatchHealthNotifications(ctx, h.notifStore, h.notifReg, h.log, notification.HealthMessage{
				Type:    string(r.Type),
				Message: r.Message,
			})
		}
	}
	h.lastIssues = currentIssues
	return nil
}
```

Add `dispatchHealthNotifications` (follows same pattern as `dispatchGrabNotifications`):

```go
func dispatchHealthNotifications(
	ctx context.Context,
	store notification.InstanceStore,
	reg *notification.Registry,
	log *slog.Logger,
	msg notification.HealthMessage,
) {
	instances, err := store.List(ctx)
	if err != nil {
		log.Error("health notification dispatch: list instances", slog.String("err", err.Error()))
		return
	}
	for _, inst := range instances {
		if !inst.OnHealthIssue {
			continue
		}
		factory, err := reg.Get(inst.Implementation)
		if err != nil {
			continue
		}
		provider := factory()
		if err := provider.OnHealthIssue(ctx, msg); err != nil {
			log.Error("health notification dispatch: OnHealthIssue failed",
				slog.String("name", inst.Name),
				slog.String("err", err.Error()),
			)
		}
	}
}
```

- [ ] **Step 5: Add checker to App struct**

Add `checker *health.Checker` to the `App` struct.

- [ ] **Step 6: Build and run full test suite**

Run: `go build ./... && go test ./...`
Expected: All pass

- [ ] **Step 7: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): wire health checker, scheduled task, and notification dispatch"
```

---

### Task 6: README Update

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update milestone counter**

Change `**Milestone 17 of 24 complete**` to `**Milestone 18 of 24 complete**`.

- [ ] **Step 2: Add health checks bullet**

After the TVDB caching bullet:

```
- **Health checks** — framework with 5 checks (database, root folders, indexers, download clients, metadata source); runs on startup and every 30 minutes; dispatches notifications for new issues
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README to reflect M18 progress"
```
