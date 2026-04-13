# Milestone 10 — Filesystem Watcher

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add event-driven filesystem monitoring so the system detects changes in series library folders and download folders without polling. When a file appears in a series folder (manual import or external tool), the system scans it. When a download completes, the watcher triggers import automatically. This replaces Sonarr's 12-hour blanket refresh with targeted, instant detection.

**Architecture:** `internal/fswatcher/` wraps `github.com/fsnotify/fsnotify` with debouncing and path-to-series resolution. One watcher per root folder. Events coalesce within a 2-second window, then enqueue targeted `ScanSeriesFolder` commands. A `RefreshMonitoredDownloads` command polls download clients for completed items and triggers `ProcessDownload`.

---

## Layout

```
internal/
├── fswatcher/
│   ├── watcher.go           # Watcher: manages fsnotify per root folder
│   ├── debouncer.go         # Debounces events by path prefix
│   └── watcher_test.go
├── commands/handlers/
│   ├── scan_series.go       # ScanSeriesFolder handler
│   ├── scan_series_test.go
│   ├── refresh_downloads.go # RefreshMonitoredDownloads handler
│   └── refresh_downloads_test.go
└── app/
    └── app.go               # Wire watcher + handlers
```

One new dependency: `github.com/fsnotify/fsnotify`.

---

## Task 1 — fsnotify dependency + Watcher skeleton

Add `fsnotify` and create the watcher with debouncing.

### watcher.go

```go
package fswatcher

import (
    "context"
    "log/slog"
    "path/filepath"
    "sync"
    "time"

    "github.com/fsnotify/fsnotify"
)

// SeriesResolver maps a filesystem path to a series ID.
type SeriesResolver interface {
    ResolveSeriesID(path string) (int64, bool)
}

// CommandEnqueuer enqueues commands by name.
type CommandEnqueuer interface {
    Enqueue(ctx context.Context, name string, body []byte) error
}

// Watcher monitors root folders for filesystem changes and enqueues
// targeted ScanSeriesFolder commands when files change.
type Watcher struct {
    resolver  SeriesResolver
    enqueuer  CommandEnqueuer
    log       *slog.Logger
    debounce  time.Duration
    mu        sync.Mutex
    watchers  map[string]*fsnotify.Watcher // root path → watcher
    cancel    context.CancelFunc
    wg        sync.WaitGroup
}

func New(resolver SeriesResolver, enqueuer CommandEnqueuer, log *slog.Logger) *Watcher

// AddRoot starts watching a root folder and all series subdirectories.
func (w *Watcher) AddRoot(rootPath string) error

// RemoveRoot stops watching a root folder.
func (w *Watcher) RemoveRoot(rootPath string) error

// Stop shuts down all watchers.
func (w *Watcher) Stop()
```

### debouncer.go

Coalesces filesystem events within a window so rapid writes (e.g., a file being copied in chunks) produce one scan, not dozens:

```go
type debouncer struct {
    mu      sync.Mutex
    pending map[string]*time.Timer // path → timer
    fire    func(path string)
    delay   time.Duration
}

func newDebouncer(delay time.Duration, fire func(path string)) *debouncer

// Trigger schedules a fire for the given path. If a timer is already
// pending for this path, it's reset.
func (d *debouncer) Trigger(path string)
```

When a timer fires, the watcher resolves the path to a series ID and enqueues `ScanSeriesFolder`.

### Tests

- `TestDebouncerCoalesces` — trigger 5 times for the same path within 100ms, verify fire called once after debounce delay
- `TestDebouncerDifferentPaths` — trigger 2 different paths, verify both fire independently
- `TestWatcherDetectsNewFile` — create a watched temp dir, write a file, verify a command was enqueued (use a recording CommandEnqueuer stub)

### Steps

- [ ] `go get github.com/fsnotify/fsnotify@v1.7.0`
- [ ] Implement watcher + debouncer
- [ ] Tests
- [ ] Commit: `feat(fswatcher): add filesystem watcher with debounced event coalescing`

---

## Task 2 — ScanSeriesFolder handler

A command handler that scans a single series folder for new/changed files and imports them.

### scan_series.go

```go
type ScanSeriesFolderHandler struct {
    library   *library.Library
    importSvc *importer.Service
    log       *slog.Logger
}

func (h *ScanSeriesFolderHandler) Handle(ctx context.Context, cmd commands.Command) error {
    var body struct {
        SeriesID int64 `json:"seriesId"`
    }
    json.Unmarshal(cmd.Body, &body)

    series, err := h.library.Series.Get(ctx, body.SeriesID)
    if err != nil { return err }

    // Scan the series path for media files not yet tracked.
    return h.importSvc.ProcessFolder(ctx, series.Path, series.ID, "")
}
```

Test: create series with a path pointing to a temp dir, put a .mkv there, run handler, verify imported.

Commit: `feat(commands/handlers): add ScanSeriesFolder handler`

---

## Task 3 — RefreshMonitoredDownloads handler

Polls download clients for completed items and triggers ProcessDownload for each.

### refresh_downloads.go

```go
type RefreshMonitoredDownloadsHandler struct {
    dcStore    downloadclient.InstanceStore
    dcRegistry *downloadclient.Registry
    cmdQueue   commands.Queue
    log        *slog.Logger
}

func (h *RefreshMonitoredDownloadsHandler) Handle(ctx context.Context, cmd commands.Command) error {
    // 1. List enabled download clients
    // 2. For each: instantiate, call Items(ctx)
    // 3. For each completed item:
    //    a. Check if a ProcessDownload command is already queued for this download ID
    //    b. If not, enqueue ProcessDownload with {downloadFolder, seriesID, downloadID}
    //       (seriesID resolved by matching item.Title against library)
}
```

The seriesID resolution uses the same `libraryLookup` pattern from M8's rsssync. For M10, we can match by looking for a history "grabbed" entry with the same download_id to find the original series.

Register as a 1-minute scheduled task (design spec calls for this).

Test: stub download client returns 1 completed item with a matching grab history entry, verify ProcessDownload enqueued.

Commit: `feat(commands/handlers): add RefreshMonitoredDownloads handler`

---

## Task 4 — Wire into app + README + push

1. Wire:
   - Create filesystem watcher in app.New
   - Add series root folders to the watcher after library init
   - Register `ScanSeriesFolder` and `RefreshMonitoredDownloads` handlers
   - Schedule `RefreshMonitoredDownloads` at 1-minute interval
   - Start watcher in Run, stop in shutdown

2. **SeriesResolver implementation** in app.go — loads all series paths and maps path prefixes to series IDs:
```go
type appSeriesResolver struct {
    library *library.Library
}

func (r *appSeriesResolver) ResolveSeriesID(path string) (int64, bool) {
    // Match the path against known series paths.
    all, _ := r.library.Series.List(context.Background())
    for _, s := range all {
        if strings.HasPrefix(path, s.Path) {
            return s.ID, true
        }
    }
    return 0, false
}
```

3. **CommandEnqueuer adapter** — wraps commands.Queue:
```go
type queueEnqueuer struct {
    queue commands.Queue
}

func (q *queueEnqueuer) Enqueue(ctx context.Context, name string, body []byte) error {
    _, err := q.queue.Enqueue(ctx, name, body, commands.PriorityNormal, commands.TriggerScheduled, "")
    return err
}
```

4. Update README: bump to M10, add filesystem watcher and download monitoring to implemented list
5. Final: tidy, lint, test, build, smoke, push, CI

---

## Done

After Task 4, the system reacts to filesystem changes in real-time (no 12-hour polling) and automatically imports completed downloads. The full automated loop is:

```
RSS Sync → Grab → Download → RefreshMonitoredDownloads → ProcessDownload → Import
                                                                              ↓
Series Folder Change → fsnotify → ScanSeriesFolder → Import (for manual adds)
```

M11 is the Sonarr v3 API compatibility layer — exposing all this functionality via REST endpoints.
