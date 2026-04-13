# M19 — Housekeeping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified daily housekeeping task that trims history, cleans orphan episode files, compacts SQLite, and recalculates series statistics.

**Architecture:** A `housekeeping.Runner` struct with methods for each cleanup operation. A `Housekeeping` command handler runs all operations on a 24-hour schedule. New store methods (`DeleteBefore` on history, `Vacuum` on db) are added to support the operations.

**Tech Stack:** Go stdlib only

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Modify | `internal/history/history.go` | Add `DeleteBefore` to Store interface |
| Modify | `internal/history/history_sqlite.go` | Implement DeleteBefore for SQLite |
| Modify | `internal/history/history_postgres.go` | Implement DeleteBefore for Postgres |
| Modify | `internal/db/sqlite.go` | Add Vacuum method |
| Modify | `internal/db/postgres.go` | Add Vacuum method (no-op) |
| Modify | `internal/config/config.go` | Add HistoryRetention config field |
| Modify | `internal/config/config_test.go` | Test history retention config |
| Create | `internal/housekeeping/housekeeping.go` | Runner with all cleanup operations |
| Create | `internal/housekeeping/housekeeping_test.go` | Tests for each operation |
| Modify | `internal/app/app.go` | Wire Runner, register handler + scheduled task |
| Modify | `README.md` | Update for M19 |

---

### Task 1: Store Methods (DeleteBefore + Vacuum)

**Files:**
- Modify: `internal/history/history.go` — add `DeleteBefore(ctx, before time.Time) (int64, error)` to Store interface
- Modify: `internal/history/history_sqlite.go` — implement DeleteBefore
- Modify: `internal/history/history_postgres.go` — implement DeleteBefore
- Modify: `internal/db/sqlite.go` — add `Vacuum(ctx) error` method
- Modify: `internal/db/postgres.go` — add `Vacuum(ctx) error` method (no-op)

- [ ] **Step 1: Add DeleteBefore to history Store interface**

In `internal/history/history.go`, add to the Store interface before the closing brace:

```go
// DeleteBefore removes all history entries with a date before the given time.
// Returns the number of deleted entries.
DeleteBefore(ctx context.Context, before time.Time) (int64, error)
```

- [ ] **Step 2: Implement DeleteBefore for SQLite**

In `internal/history/history_sqlite.go`, add:

```go
func (s *sqliteStore) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.pool.DB().ExecContext(ctx, "DELETE FROM history WHERE date < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
```

- [ ] **Step 3: Implement DeleteBefore for Postgres**

In `internal/history/history_postgres.go`, add:

```go
func (s *postgresStore) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Pool().Exec(ctx, "DELETE FROM history WHERE date < $1", before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
```

- [ ] **Step 4: Add Vacuum to SQLite pool**

In `internal/db/sqlite.go`, add:

```go
// Vacuum compacts the SQLite database file by rebuilding it.
func (p *SQLitePool) Vacuum(ctx context.Context) error {
	_, err := p.DB().ExecContext(ctx, "VACUUM")
	return err
}
```

- [ ] **Step 5: Add Vacuum to Postgres pool (no-op)**

In `internal/db/postgres.go`, add:

```go
// Vacuum is a no-op for Postgres — autovacuum handles compaction.
func (p *PostgresPool) Vacuum(_ context.Context) error {
	return nil
}
```

- [ ] **Step 6: Run tests**

Run: `go build ./... && go test ./internal/history/ ./internal/db/ -v`

- [ ] **Step 7: Commit**

```bash
git commit -m "feat(db,history): add DeleteBefore and Vacuum methods for housekeeping"
```

---

### Task 2: History Retention Config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Add HistoryRetention to Config**

Add `HistoryRetention time.Duration \`yaml:"history_retention"\`` to the top-level `Config` struct.

In `Default()`, add: `HistoryRetention: 90 * 24 * time.Hour, // 90 days`

In `Load()`, add env var parsing after the TVDB block:

```go
if v := getenv("SONARR2_HISTORY_RETENTION"); v != "" {
	d, err := time.ParseDuration(v)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_HISTORY_RETENTION must be a duration, got %q: %w", v, err)
	}
	cfg.HistoryRetention = d
}
```

- [ ] **Step 2: Add tests**

```go
func TestHistoryRetentionDefault(t *testing.T) {
	cfg := Default()
	if cfg.HistoryRetention != 90*24*time.Hour {
		t.Errorf("default HistoryRetention = %v, want 2160h", cfg.HistoryRetention)
	}
}

func TestHistoryRetentionEnvOverride(t *testing.T) {
	env := map[string]string{"SONARR2_HISTORY_RETENTION": "720h"}
	cfg, err := Load(nil, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HistoryRetention != 720*time.Hour {
		t.Errorf("HistoryRetention = %v, want 720h", cfg.HistoryRetention)
	}
}
```

- [ ] **Step 3: Run tests and commit**

```bash
git commit -m "feat(config): add history retention configuration"
```

---

### Task 3: Housekeeping Runner

**Files:**
- Create: `internal/housekeeping/housekeeping.go`
- Create: `internal/housekeeping/housekeeping_test.go`

- [ ] **Step 1: Write failing tests**

Tests with mock interfaces:

- `TestTrimHistory` — mock HistoryTrimmer, verify DeleteBefore called with correct cutoff
- `TestCleanOrphanEpisodeFiles` — mock that returns files, some exist some don't, verify only missing deleted
- `TestVacuumSQLite` — mock Vacuumer, verify called
- `TestRecalculateStatistics` — mock series lister + stats recomputer, verify all series recomputed
- `TestRunAll` — all mocks, verify all operations run, one failure doesn't stop others

- [ ] **Step 2: Implement Runner**

```go
package housekeeping

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// HistoryTrimmer deletes old history entries.
type HistoryTrimmer interface {
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// EpisodeFileLister lists episode files for a series.
type EpisodeFileLister interface {
	ListForSeries(ctx context.Context, seriesID int64) ([]EpisodeFileInfo, error)
	Delete(ctx context.Context, id int64) error
}

// EpisodeFileInfo is the subset of library.EpisodeFile needed.
type EpisodeFileInfo struct {
	ID           int64
	SeriesID     int64
	RelativePath string
}

// SeriesLister lists all series for iteration.
type SeriesLister interface {
	ListAll(ctx context.Context) ([]SeriesInfo, error)
}

// SeriesInfo is the subset of library.Series needed.
type SeriesInfo struct {
	ID   int64
	Path string
}

// StatsRecomputer recalculates statistics for a series.
type StatsRecomputer interface {
	Recompute(ctx context.Context, seriesID int64) error
}

// Vacuumer compacts the database.
type Vacuumer interface {
	Vacuum(ctx context.Context) error
}

// Options configures the housekeeping Runner.
type Options struct {
	History          HistoryTrimmer
	EpisodeFiles     EpisodeFileLister
	Series           SeriesLister
	Stats            StatsRecomputer
	DB               Vacuumer
	Log              *slog.Logger
	HistoryRetention time.Duration
}

// Runner performs all housekeeping operations.
type Runner struct {
	opts Options
}

// New creates a Runner.
func New(opts Options) *Runner {
	if opts.HistoryRetention <= 0 {
		opts.HistoryRetention = 90 * 24 * time.Hour
	}
	return &Runner{opts: opts}
}

// Run executes all housekeeping operations. Errors are logged but non-fatal.
func (r *Runner) Run(ctx context.Context) {
	r.trimHistory(ctx)
	r.cleanOrphanEpisodeFiles(ctx)
	r.recalculateStatistics(ctx)
	r.vacuumDatabase(ctx)
}

func (r *Runner) trimHistory(ctx context.Context) {
	cutoff := time.Now().Add(-r.opts.HistoryRetention)
	n, err := r.opts.History.DeleteBefore(ctx, cutoff)
	if err != nil {
		r.opts.Log.Error("housekeeping: trim history", slog.String("err", err.Error()))
		return
	}
	if n > 0 {
		r.opts.Log.Info("housekeeping: trimmed history", slog.Int64("deleted", n))
	}
}

func (r *Runner) cleanOrphanEpisodeFiles(ctx context.Context) {
	series, err := r.opts.Series.ListAll(ctx)
	if err != nil {
		r.opts.Log.Error("housekeeping: list series", slog.String("err", err.Error()))
		return
	}
	var deleted int
	for _, s := range series {
		files, err := r.opts.EpisodeFiles.ListForSeries(ctx, s.ID)
		if err != nil {
			r.opts.Log.Error("housekeeping: list episode files",
				slog.Int64("seriesID", s.ID), slog.String("err", err.Error()))
			continue
		}
		for _, f := range files {
			fullPath := filepath.Join(s.Path, f.RelativePath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				if err := r.opts.EpisodeFiles.Delete(ctx, f.ID); err != nil {
					r.opts.Log.Error("housekeeping: delete orphan file",
						slog.Int64("id", f.ID), slog.String("err", err.Error()))
					continue
				}
				deleted++
			}
		}
	}
	if deleted > 0 {
		r.opts.Log.Info("housekeeping: cleaned orphan episode files", slog.Int("deleted", deleted))
	}
}

func (r *Runner) recalculateStatistics(ctx context.Context) {
	series, err := r.opts.Series.ListAll(ctx)
	if err != nil {
		r.opts.Log.Error("housekeeping: list series for stats", slog.String("err", err.Error()))
		return
	}
	for _, s := range series {
		if err := r.opts.Stats.Recompute(ctx, s.ID); err != nil {
			r.opts.Log.Error("housekeeping: recompute stats",
				slog.Int64("seriesID", s.ID), slog.String("err", err.Error()))
		}
	}
}

func (r *Runner) vacuumDatabase(ctx context.Context) {
	if err := r.opts.DB.Vacuum(ctx); err != nil {
		r.opts.Log.Error("housekeeping: vacuum", slog.String("err", err.Error()))
	}
}
```

- [ ] **Step 3: Run tests and commit**

```bash
git commit -m "feat(housekeeping): add Runner with history trim, orphan cleanup, vacuum, stats recalc"
```

---

### Task 4: Wire in app.go + README

**Files:**
- Modify: `internal/app/app.go`
- Modify: `README.md`

- [ ] **Step 1: Add import and create Runner**

Add `"github.com/ajthom90/sonarr2/internal/housekeeping"` import.

After the health check block, create the Runner and register the handler:

```go
// Housekeeping runner.
housekeeper := housekeeping.New(housekeeping.Options{
	History:          histStore,
	EpisodeFiles:     &episodeFileAdapter{store: lib.EpisodeFiles},
	Series:           &seriesAdapter{store: lib.Series},
	Stats:            lib.Stats,
	DB:               &vacuumAdapter{pool: pool},
	Log:              log,
	HistoryRetention: cfg.HistoryRetention,
})

housekeepingHandler := &housekeepingTaskHandler{runner: housekeeper}
reg.Register("Housekeeping", housekeepingHandler)

// Schedule Housekeeping at 24-hour interval.
if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
	TypeName:      "Housekeeping",
	IntervalSecs:  86400,
	NextExecution: time.Now().Add(24 * time.Hour),
}); err != nil {
	_ = pool.Close()
	return nil, fmt.Errorf("app: upsert Housekeeping task: %w", err)
}
```

- [ ] **Step 2: Add adapter types**

```go
// episodeFileAdapter adapts library.EpisodeFilesStore for housekeeping.
type episodeFileAdapter struct {
	store library.EpisodeFilesStore
}

func (a *episodeFileAdapter) ListForSeries(ctx context.Context, seriesID int64) ([]housekeeping.EpisodeFileInfo, error) {
	files, err := a.store.ListForSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	result := make([]housekeeping.EpisodeFileInfo, len(files))
	for i, f := range files {
		result[i] = housekeeping.EpisodeFileInfo{
			ID:           f.ID,
			SeriesID:     f.SeriesID,
			RelativePath: f.RelativePath,
		}
	}
	return result, nil
}

func (a *episodeFileAdapter) Delete(ctx context.Context, id int64) error {
	return a.store.Delete(ctx, id)
}

// seriesAdapter adapts library.SeriesStore for housekeeping.
type seriesAdapter struct {
	store library.SeriesStore
}

func (a *seriesAdapter) ListAll(ctx context.Context) ([]housekeeping.SeriesInfo, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]housekeeping.SeriesInfo, len(all))
	for i, s := range all {
		result[i] = housekeeping.SeriesInfo{ID: s.ID, Path: s.Path}
	}
	return result, nil
}

// vacuumAdapter adapts db.Pool for housekeeping.Vacuumer.
type vacuumAdapter struct {
	pool db.Pool
}

func (a *vacuumAdapter) Vacuum(ctx context.Context) error {
	switch p := a.pool.(type) {
	case *db.SQLitePool:
		return p.Vacuum(ctx)
	case *db.PostgresPool:
		return p.Vacuum(ctx)
	default:
		return nil
	}
}

// housekeepingTaskHandler wraps the Runner as a command handler.
type housekeepingTaskHandler struct {
	runner *housekeeping.Runner
}

func (h *housekeepingTaskHandler) Handle(ctx context.Context, _ commands.Command) error {
	h.runner.Run(ctx)
	return nil
}
```

- [ ] **Step 3: Update README**

Bump milestone to 19, add housekeeping bullet.

- [ ] **Step 4: Build, test, commit**

```bash
git commit -m "feat(app): wire housekeeping runner with scheduled task"
git commit -m "docs: update README to reflect M19 progress"
```
