# M18 — Health Checks

## Overview

Add a health check framework that runs checks on startup and periodically, exposes results via the `/health` API endpoint, and dispatches notifications for health issues. The frontend already consumes the `/health` endpoint (currently a stub returning `[]`).

## Architecture

### Health Check Framework

New package `internal/health/` with:

```go
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
    Type   Level  `json:"type"`
    Message string `json:"message"`
    WikiURL string `json:"wikiUrl,omitempty"`
}

// Check is a single health check that evaluates one aspect of system health.
type Check interface {
    // Name returns a unique identifier for this check (e.g., "IndexerCheck").
    Name() string
    // Check runs the health check and returns zero or more results.
    // An empty slice means healthy.
    Check(ctx context.Context) []Result
}
```

### Checker (Runner)

```go
// Checker runs all registered health checks and aggregates results.
type Checker struct {
    checks []Check
    mu     sync.RWMutex
    last   []Result // most recent results
}

func NewChecker(checks ...Check) *Checker
func (c *Checker) RunAll(ctx context.Context) []Result  // runs all checks, caches results
func (c *Checker) Results() []Result                     // returns cached results (read-only)
```

`RunAll` runs all checks sequentially (health checks are fast — no need for concurrency), stores results, and returns them. `Results` returns the last cached results for the API to serve without re-running.

### Health Checks (Initial Set)

These cover the most valuable checks for a running instance:

| Check | Source Name | What it checks | Level |
|---|---|---|---|
| DatabaseCheck | DatabaseCheck | `pool.Ping(ctx)` succeeds | error |
| RootFolderCheck | RootFolderCheck | Each series' root folder exists and is writable | warning per missing folder |
| IndexerCheck | IndexerCheck | At least one indexer is enabled | warning |
| DownloadClientCheck | DownloadClientCheck | At least one download client is enabled | warning |
| MetadataSourceCheck | MetadataSourceCheck | TVDB API key is configured (non-empty) | warning |

Each check is a separate file in `internal/health/checks/` implementing `health.Check`.

### API Integration

**V3** — Replace the stub in `internal/api/v3/health.go`:
- `GET /api/v3/health` returns `[]Result` from `checker.Results()`
- Wire `*health.Checker` into the API `Deps` struct

**V6** — Add `internal/api/v6/health.go`:
- `GET /api/v6/health` returns the same `[]Result`

### Notification Dispatch

After `RunAll` completes, compare new results to previous results. For any newly-appeared issues (not in the previous set), dispatch `OnHealthIssue` to enabled notification providers. This avoids re-notifying on persistent issues every check cycle.

### Scheduled Execution

Register a `HealthCheck` scheduled task in the scheduler (30-minute interval). The task handler calls `checker.RunAll()` and dispatches notifications for new issues.

Also run once at startup (inside `app.New` after all stores are wired).

## Dependencies

Each check needs access to specific stores/config:

| Check | Dependencies |
|---|---|
| DatabaseCheck | `db.Pool` |
| RootFolderCheck | `library.SeriesStore` |
| IndexerCheck | `indexer.InstanceStore` |
| DownloadClientCheck | `downloadclient.InstanceStore` |
| MetadataSourceCheck | `config.TVDBConfig` |

These are injected via constructors — no global state.

## Wiring in app.go

```go
checker := health.NewChecker(
    checks.NewDatabaseCheck(pool),
    checks.NewRootFolderCheck(lib.Series),
    checks.NewIndexerCheck(idxStore),
    checks.NewDownloadClientCheck(dcStore),
    checks.NewMetadataSourceCheck(cfg.TVDB),
)

// Initial health check on startup.
checker.RunAll(ctx)
```

The `Checker` is added to `api.Deps` so both v3 and v6 endpoints can serve results.

## Testing

- **Each check**: unit test with mock dependencies (e.g., mock pool that returns error on Ping → DatabaseCheck returns error result)
- **Checker**: test with multiple checks, verify RunAll aggregates correctly, Results returns cached
- **API endpoints**: test v3 and v6 return JSON array matching the Result shape
- **Notification dispatch**: test that new issues trigger OnHealthIssue, persistent issues don't re-notify

## Out of Scope

- Authentication checks (M22 — ops hardening)
- Disk space monitoring (M19 — housekeeping)
- Update availability (M23 — release engineering)
- Import list checks (import lists not yet implemented)
- Proxy checks (no proxy support yet)
- Wiki URLs (can be added incrementally as wiki content is created)
