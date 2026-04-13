# M19 — Housekeeping

## Overview

Add a unified daily housekeeping task that performs database cleanup operations: trimming old history, removing orphan episode file records, compacting SQLite, and recalculating stale series statistics. Configurable retention periods via env vars.

## Architecture

### Housekeeping Package

New package `internal/housekeeping/` with individual cleanup operations as functions, and a `Runner` that executes them all.

```go
// Runner executes all housekeeping operations.
type Runner struct {
    pool            db.Pool
    historyStore    history.Store
    episodeFiles    library.EpisodeFilesStore
    series          library.SeriesStore
    stats           library.SeriesStatsStore
    log             *slog.Logger
    historyRetention time.Duration // default 90 days
}

func New(opts Options) *Runner
func (r *Runner) Run(ctx context.Context) error  // runs all cleanup ops
```

### Cleanup Operations

| Operation | What it does | Config |
|---|---|---|
| TrimHistory | Deletes history entries older than retention period | `SONARR2_HISTORY_RETENTION` (default `2160h` = 90 days) |
| CleanOrphanEpisodeFiles | Removes episode_file rows where the file path doesn't exist on disk | None |
| VacuumDatabase | Runs `VACUUM` on SQLite, no-op on Postgres | None |
| RecalculateStatistics | Calls `Stats.Recompute` for every series | None |

Each operation is a method on `Runner` that logs what it did and returns an error (logged but non-fatal — one failure doesn't stop others).

### Store Methods Needed

**history.Store** — needs `DeleteBefore(ctx, cutoff time.Time) (int64, error)` to delete entries older than cutoff. Returns count of deleted rows.

**library.EpisodeFilesStore** — needs `ListAll(ctx) ([]EpisodeFile, error)` to list all episode files for orphan checking. Already has `Delete(ctx, id)`.

**library.SeriesStore** — already has `List(ctx)` for iterating series.

**library.SeriesStatsStore** — already has `Recompute(ctx, seriesID)`.

**db.Pool** — needs `Vacuum(ctx) error` method. SQLite implementation runs `VACUUM`, Postgres returns nil.

### Scheduled Task

Register `Housekeeping` as a scheduled task with a 24-hour interval. The `MessagingCleanup` task (1-hour) remains separate since it runs more frequently.

### Configuration

| Variable | Default | Description |
|---|---|---|
| `SONARR2_HISTORY_RETENTION` | `2160h` | How long to keep history entries (90 days) |

## Wiring in app.go

```go
housekeeper := housekeeping.New(housekeeping.Options{
    Pool:             pool,
    HistoryStore:     histStore,
    EpisodeFiles:     lib.EpisodeFiles,
    Series:           lib.Series,
    Stats:            lib.Stats,
    Log:              log,
    HistoryRetention: cfg.HistoryRetention,
})

housekeepingHandler := &housekeepingTaskHandler{runner: housekeeper}
reg.Register("Housekeeping", housekeepingHandler)
```

## Testing

- **TrimHistory**: mock store, verify DeleteBefore called with correct cutoff
- **CleanOrphanEpisodeFiles**: mock store returning files, create some real files with t.TempDir(), verify only missing ones are deleted
- **VacuumDatabase**: verify called on SQLite pool (mock), verify no-op on Postgres
- **RecalculateStatistics**: mock series list + stats recompute, verify all series recomputed
- **Runner.Run**: verify all operations called, one failure doesn't stop others

## Out of Scope

- Blocklist cleanup (no blocklist feature)
- Metadata/subtitle/extra file rows (no such tables)
- Recycle bin (no recycle bin feature)
- Media cover files (no cover storage)
- Log entry trimming (no log_entries table)
- 3 AM scheduling (scheduler doesn't support time-of-day targeting; uses interval)
