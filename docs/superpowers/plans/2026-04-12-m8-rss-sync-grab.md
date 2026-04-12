# Milestone 8 — RSS Sync + Grab Flow

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire the first end-to-end "grab a release" pipeline: RSS feed → parse titles → match to series → evaluate against decision engine → pick the best → send to download client → record in history. After M8, the system automatically monitors indexer RSS feeds on a schedule and grabs releases that match monitored series.

**Architecture:** `internal/rsssync/` owns the RSS sync command handler. It pulls releases from all enabled indexers, parses each title, matches to a known series, runs the decision engine, and grabs the top-ranked release via the download client. A new `history` table records grab events so the `AlreadyImported` spec can check for duplicates.

---

## Layout

```
internal/
├── rsssync/
│   ├── rsssync.go           # RssSyncHandler command handler
│   └── rsssync_test.go
├── grab/
│   ├── grab.go              # GrabService: pick download client + send
│   └── grab_test.go
├── history/
│   ├── history.go           # HistoryEntry type + HistoryStore interface
│   ├── history_postgres.go
│   ├── history_sqlite.go
│   └── history_test.go
└── db/
    ├── migrations/{postgres,sqlite}/00014_history.sql
    ├── queries/{postgres,sqlite}/history.sql
    └── gen/ (regenerated)
```

---

## Task 1 — History table + store

Add a `history` table that records grab/import/fail events for episodes.

### Postgres 00014_history.sql

```sql
-- +goose Up
CREATE TABLE history (
    id            BIGSERIAL PRIMARY KEY,
    episode_id    BIGINT NOT NULL,
    series_id     BIGINT NOT NULL,
    source_title  TEXT NOT NULL,
    quality_name  TEXT NOT NULL DEFAULT '',
    event_type    TEXT NOT NULL,
    date          TIMESTAMPTZ NOT NULL DEFAULT now(),
    download_id   TEXT NOT NULL DEFAULT '',
    data          JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX history_series_date_idx ON history (series_id, date DESC);
CREATE INDEX history_episode_date_idx ON history (episode_id, date DESC);
CREATE INDEX history_download_id_idx ON history (download_id) WHERE download_id != '';

-- +goose Down
DROP TABLE history;
```

SQLite variant with usual adaptations.

Queries: `CreateHistoryEntry :one`, `ListForSeries :many`, `ListForEpisode :many`, `FindByDownloadID :many`, `DeleteForSeries :exec`.

### HistoryStore interface

```go
type EventType string
const (
    EventGrabbed          EventType = "grabbed"
    EventDownloadImported EventType = "downloadImported"
    EventDownloadFailed   EventType = "downloadFailed"
    EventEpisodeRenamed   EventType = "episodeRenamed"
    EventEpisodeDeleted   EventType = "episodeFileDeleted"
)

type HistoryEntry struct {
    ID          int64
    EpisodeID   int64
    SeriesID    int64
    SourceTitle string
    QualityName string
    EventType   EventType
    Date        time.Time
    DownloadID  string
    Data        json.RawMessage
}

type HistoryStore interface {
    Create(ctx, HistoryEntry) (HistoryEntry, error)
    ListForSeries(ctx, seriesID int64) ([]HistoryEntry, error)
    ListForEpisode(ctx, episodeID int64) ([]HistoryEntry, error)
    FindByDownloadID(ctx, downloadID string) ([]HistoryEntry, error)
    DeleteForSeries(ctx, seriesID int64) error
}
```

### Steps

- [ ] Create migration + query files + sqlc regen
- [ ] Implement HistoryStore for both dialects
- [ ] CRUD tests (SQLite in-memory)
- [ ] Commit: `feat(history): add history table and store for grab/import events`

---

## Task 2 — GrabService

A service that picks the right download client and sends a release to it.

### grab.go

```go
package grab

type Service struct {
    dcStore    downloadclient.InstanceStore
    dcRegistry *downloadclient.Registry
    history    history.HistoryStore
    bus        events.Bus
    log        *slog.Logger
}

func New(dcStore downloadclient.InstanceStore, dcRegistry *downloadclient.Registry,
    history history.HistoryStore, bus events.Bus, log *slog.Logger) *Service

// Grab sends a release to the appropriate download client and records
// a history entry.
func (s *Service) Grab(ctx context.Context, release indexer.Release,
    seriesID int64, episodeIDs []int64, qualityName string) error {
    // 1. List enabled download clients matching the release's protocol
    // 2. Pick the highest-priority one
    // 3. Instantiate it via the registry factory + stored settings
    // 4. Call dc.Add(ctx, release.DownloadURL, release.Title)
    // 5. Record a "grabbed" history entry with the download ID
    // 6. Publish a ReleasesGrabbed event
}
```

### Tests

Use stubs for dcStore, dcRegistry, history. Verify:
- `TestGrabSuccess` — release sent to client, history recorded, event published
- `TestGrabNoEnabledClient` — returns error when no matching client
- `TestGrabClientAddFails` — client.Add errors, grab returns error, no history recorded

### Steps

- [ ] Write tests
- [ ] Implement GrabService
- [ ] Commit: `feat(grab): add GrabService for sending releases to download clients`

---

## Task 3 — RssSyncHandler

The main RSS sync command handler — the heart of M8.

### rsssync.go

```go
package rsssync

type Handler struct {
    idxStore      indexer.InstanceStore
    idxRegistry   *indexer.Registry
    library       *library.Library
    engine        *decisionengine.Engine
    grabService   *grab.Service
    qualityDefs   profiles.QualityDefinitionStore
    qualityProfs  profiles.QualityProfileStore
    cfStore       customformats.Store
    parser        parserPkg   // references internal/parser
    log           *slog.Logger
}

func (h *Handler) Handle(ctx context.Context, cmd commands.Command) error {
    // 1. List all enabled indexers with RSS enabled
    // 2. For each indexer:
    //    a. Instantiate via registry + settings
    //    b. Call FetchRss(ctx) → []Release
    // 3. For each release:
    //    a. ParseTitle(release.Title) → ParsedEpisodeInfo
    //    b. Match to a series via library lookup
    //    c. If no match, skip
    //    d. Find the episodes by (seriesID, season, episode)
    //    e. If episodes not monitored, skip
    //    f. Look up the series' quality profile
    //    g. Score custom formats
    //    h. Build a RemoteEpisode
    //    i. Run decision engine Evaluate
    //    j. If rejected, log reasons and skip
    // 4. Collect all accepted RemoteEpisodes
    // 5. Group by (seriesID, season, episode) — pick the best per group via Rank
    // 6. For each winner: call grabService.Grab
    // 7. Return nil (individual grab errors are logged but don't fail the sync)
}
```

### SeriesLookup implementation

The parser's `SeriesLookup` interface needs an implementation backed by the library. Create a thin adapter:

```go
type libraryLookup struct {
    series library.SeriesStore
}

func (l *libraryLookup) FindByTitle(ctx context.Context, title string) (int64, bool, error) {
    // Try exact slug match first, then title match.
    // For M8, use a simple case-insensitive title comparison.
    // Full fuzzy matching comes in a later milestone.
    allSeries, err := l.series.List(ctx)
    if err != nil {
        return 0, false, err
    }
    normalized := strings.ToLower(strings.TrimSpace(title))
    for _, s := range allSeries {
        if strings.ToLower(s.Title) == normalized || strings.ToLower(s.Slug) == normalized {
            return s.ID, true, nil
        }
    }
    return 0, false, nil
}
```

### Tests

Use stubs/fakes throughout — no real HTTP, no real DB beyond in-memory SQLite:

- `TestRssSyncMatchesAndGrabs` — stub indexer returns 2 releases, one matches a series, decision engine accepts it, verify grab called once
- `TestRssSyncSkipsUnmatchedReleases` — stub returns releases that don't match any series, verify no grabs
- `TestRssSyncSkipsRejectedReleases` — decision engine rejects the release, verify no grab
- `TestRssSyncPicksBestRelease` — two releases for same episode, verify only the higher-ranked one is grabbed

### Steps

- [ ] Write stubs and tests
- [ ] Implement RssSyncHandler + libraryLookup
- [ ] Commit: `feat(rsssync): add RSS sync handler with parse-match-evaluate-grab pipeline`

---

## Task 4 — Wire into app + schedule RSS sync

Wire everything into app.New:

1. Create HistoryStore (dialect dispatch)
2. Create GrabService
3. Create decision engine with the 8 M5 specs
4. Create RssSyncHandler with all dependencies
5. Register: `reg.Register("RssSync", rssSyncHandler)`
6. Register scheduled task: `RssSync` with 15-minute interval (the default from the design spec)

Also subscribe a `SeriesDeleted` event handler to clean up history when a series is deleted:
```go
events.SubscribeSync[library.SeriesDeleted](bus, func(ctx context.Context, e library.SeriesDeleted) error {
    return historyStore.DeleteForSeries(ctx, e.ID)
})
```

### Steps

- [ ] Wire all new components
- [ ] Add integration test: create series + episodes, enqueue RssSync with a stub indexer, verify history entry
- [ ] Commit: `feat(app): wire RSS sync, grab service, history, and schedule 15-min RSS sync`

---

## Task 5 — Update README + final verification + push

1. Update README: add "RSS sync pipeline" and "History tracking" to implemented list, bump to M8
2. `go mod tidy`, `make lint`, full test suite, build, smoke test
3. Push + CI watch

---

## Done

After Task 5, the binary has a complete grab pipeline: indexer RSS → parse → match → evaluate → rank → grab → history. This is the first feature that does something *useful* for an end user — if they configure an indexer and a download client, the system will automatically grab releases for their monitored series every 15 minutes. M9 (import pipeline) closes the loop by importing the completed downloads into the library.
