# sonarr2 — Ground-Up Rewrite Design

**Date:** 2026-04-10
**Status:** Draft (pending user review)
**Authors:** Claude (drafted), AJ Thompson (reviewing)

## 1. Overview

sonarr2 is a ground-up rewrite of [Sonarr](https://github.com/Sonarr/Sonarr) (the .NET TV-series PVR) in Go, targeting large-library performance, low-resource homelab deployments, full feature parity with upstream, and a clean MIT license.

### 1.1 Goals

- **Full feature parity** with current stable Sonarr, including every upstream indexer, download client, notification, import list, and metadata consumer — see §7 and Appendix A.
- **Drop-in replacement at the wire level**: implement Sonarr's `/api/v3/*` REST surface and SignalR real-time protocol so existing clients (Overseerr, Jellyseerr, Prowlarr, LunaSea, nzb360, Notifiarr, Tautulli, Radarr cross-integration, Home Assistant) work without modification.
- **Migration tool**: one-shot import of an existing Sonarr SQLite or Postgres database (Sonarr v4.x and v5.x).
- **Performance targets** on a 10,000-series / 500,000-episode-file reference library running on 2 cores / 4 GB RAM — see §6.11.
- **MIT licensed**, via clean-room reimplementation (no source code copied from Sonarr).
- **Multi-arch Docker**: `linux/amd64`, `linux/arm64`, `linux/arm/v7`, plus native binaries for macOS, Linux, and Windows.

### 1.2 Non-goals (v1)

- Runtime plugin loading (out-of-process gRPC/WASM). Interfaces designed to support it later.
- Multi-user with roles. Single-user, matching Sonarr.
- Built-in HTTPS termination. Users run behind a reverse proxy.
- Clustering / multi-instance. Single-node only.
- GraphQL API. REST v3 + REST v6 is enough.
- Mobile app, Radarr/Lidarr/Readarr equivalents.
- TRaSH Guides auto-install browser. Users import JSON manually.

### 1.3 Clean-room constraint

Sonarr is GPL-3.0 licensed. sonarr2 is MIT licensed. To avoid license contamination:

- Claude does **not** read Sonarr's C# source code during design or implementation.
- Design and implementation draw only from (a) Sonarr's public OpenAPI specs (`/api/v3/openapi.json`), (b) observable wire formats and HTTP behavior, (c) documented features, (d) UI screenshots and the running application's interface.
- APIs themselves are not copyrightable (_Google v. Oracle_, 2021), so re-implementing the same REST surface is permitted. Re-implementing the same business logic from a written spec is also permitted.
- Translations, when seeded, are regenerated from Sonarr's public locale files with attribution; non-literal expressions of the same functionality are independently authored.

## 2. Technology Choices

### 2.1 Backend: Go

**Why Go (not Rust, not Kotlin, not Python):**

- **Target fit**: Go binaries are ~20-80 MB, baseline RAM ~30-100 MB. A Sonarr-equivalent managing 29k series comfortably fits under 4 GB with headroom.
- **Cross-compilation**: `GOOS=linux GOARCH=arm64 go build` produces a native static binary. Trivial multi-arch Docker.
- **Claude-authored code quality**: Go produces correct first-pass code more often than Rust in LLM-assisted workflows. Explicit, regular, minimal hidden control flow.
- **Ecosystem match**: mature libraries for every provider kind (`pgx`, `modernc.org/sqlite`, `gorilla/websocket`, `anacrolix/torrent`, `robfig/cron`, `fsnotify`, `prometheus/client_golang`, `go.opentelemetry.io/otel`).
- **Concurrency model**: goroutines + channels map cleanly to the N-independent-workers pattern a PVR needs.
- **Contributor pool**: large, overlapping with the homelab/Kubernetes/Prometheus/autobrr ecosystem.

**Rust was considered** and rejected as the primary language for the same reasons above, flipped: slower iteration cycles in an LLM-heavy authoring workflow, slower compile times, smaller contributor pool. Rust's raw performance advantage is not meaningful for an IO-bound workload.

**Kotlin (with GraalVM native-image) was considered** and rejected due to native-image fragility, heavier runtime memory footprint, and an uglier cross-compile story.

### 2.2 Frontend: React + TypeScript + Vite

Preserving Sonarr's UI structure is part of "drop-in replacement." React + TypeScript matches Sonarr's own stack, minimizing the porting gap. Vite replaces Sonarr's Webpack. Redux is replaced with **TanStack Query** (server state) and **Zustand** (local UI state) — a meaningfully cleaner architecture than upstream.

Styling uses **CSS Modules with CSS custom properties** (design tokens) rather than Tailwind, to match Sonarr's dark-data-dense aesthetic faithfully. Interactive primitives come from **Radix UI** (unstyled, a11y-compliant).

### 2.3 Database: Postgres-first, SQLite supported

- **Postgres is the primary target.** Hot paths are designed around row-level locks and MVCC. This avoids the "database is locked" class of bugs that plague upstream Sonarr on SQLite.
- **SQLite is supported** for small installs (single-user homelabs with modest libraries) via a **single-writer goroutine discipline**: all writes funnel through one goroutine fed by a channel; reads use a separate read-only connection pool with `PRAGMA query_only=1`. This makes the single-writer constraint explicit rather than inheriting SQLite's lock behavior.
- **Query layer**: `sqlc` for compile-time-checked typed SQL. Dialect-specific query files where required; shared query files where dialect-neutral SQL works.
- **Migrations**: `goose` with per-dialect subdirectories (`internal/db/migrations/postgres/` and `internal/db/migrations/sqlite/`).
- **Driver**: `pgx` / `pgxpool` for Postgres, `modernc.org/sqlite` (pure-Go, no CGo) for SQLite. Pure-Go SQLite lets us build static binaries without CGo toolchains, trading slight write-heavy performance for a much simpler multi-arch story.

## 3. Process Topology

**One Go binary, one frontend bundle, embedded together.** No separate worker process, no microservices, no Redis, no external queue.

The binary hosts:

1. **HTTP server** — REST API (`/api/v3`, `/api/v6`), static frontend (`//go:embed`), WebSocket for SignalR emulation and the new stream API, auth middleware.
2. **In-process scheduler** — ticks every ~500ms, reads `scheduled_tasks`, enqueues due commands.
3. **In-process command worker pool** — N goroutines (default `min(NumCPU, 4)`) pulling from the durable `commands` queue.
4. **In-process event bus** — typed pub/sub via `Subscribe[T]`, synchronous by default with async and ordered variants.
5. **Filesystem watcher** — one `fsnotify` watcher per root folder, feeding targeted `ScanSeriesFolder` commands.
6. **Shared outbound HTTP client** — per-host connection pooling, per-host rate limiting, retry with backoff, circuit breakers.

### 3.1 Docker image

Multi-stage build, final stage is `gcr.io/distroless/static-debian12`. Final image ~20-40 MB. Multi-arch via `buildx` (`linux/amd64,linux/arm64,linux/arm/v7`).

Postgres is **not** bundled. Users running Postgres bring their own via `docker-compose`. Users running SQLite need nothing extra. Config directory convention matches the *arr ecosystem (`/config`).

### 3.2 Repository layout

```
sonarr2/
├── cmd/sonarr/main.go
├── internal/
│   ├── app/                   # DI composition, config, lifecycle
│   ├── config/                # config file + env + flags parsing
│   ├── db/                    # pool, migrations, sqlc output
│   │   ├── migrations/{postgres,sqlite}/
│   │   └── queries/
│   ├── domain/                # series, episode, file, profile, etc.
│   ├── scheduler/             # tick loop
│   ├── commands/              # queue, worker pool, command types
│   ├── events/                # typed event bus
│   ├── parser/                # title + quality + series matching
│   ├── decisionengine/
│   │   └── specs/             # one file per rejection rule
│   ├── customformats/
│   ├── profiles/              # quality, release, delay, auto-tagging
│   ├── import/
│   │   └── specs/             # one file per import spec
│   ├── organizer/             # naming tokens, rename, move
│   ├── mediafiles/            # ffprobe / mediainfo
│   ├── mediacover/            # poster/fanart download + resize
│   ├── providers/
│   │   ├── indexer/           # one package per implementation
│   │   ├── downloadclient/
│   │   ├── notification/
│   │   ├── importlist/
│   │   ├── metadata/          # nfo/artwork writers
│   │   └── metadatasource/    # TVDB / TMDb / TVMaze
│   ├── health/                # health checks
│   ├── housekeeping/          # orphan cleanup, trim, vacuum
│   ├── backup/
│   ├── update/
│   ├── auth/
│   ├── api/
│   │   ├── v3/                # Sonarr v3 compat surface
│   │   └── v6/                # fresh API surface
│   ├── realtime/              # SignalR emulation + SSE
│   ├── httpclient/
│   ├── logging/
│   └── buildinfo/             # version via -ldflags
├── frontend/
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── tools/
│   └── sonarr-migrate/        # standalone migration binary
├── docs/
│   └── superpowers/specs/     # design docs
├── docker/
│   ├── Dockerfile
│   └── entrypoint.sh
├── .github/workflows/
├── go.mod
├── LICENSE                    # MIT
└── README.md
```

`internal/` enforces package privacy at the Go compiler level. Tools under `tools/` can import from `internal/` because they're in the same module.

## 4. Domain Model and Storage

### 4.1 Entity groups

**Media catalog**
- `series` — tvdb/tmdb/imdb IDs, title, slug, year, status, network, runtime, air_time, series_type (standard/daily/anime), path, root_folder_id, quality_profile_id, monitored, monitor_new_items, season_folder, use_scene_numbering, first_aired, last_aired, original_language, original_country, genres, ratings, added, tags.
- `seasons` — `(series_id, season_number, monitored)`. Separate table so monitored flips don't rewrite the parent.
- `episodes` — series_id, season_number, episode_number, absolute_episode_number, scene_* numbers, title, overview, air_date, air_date_utc, runtime, tvdb_id, imdb_id, monitored, unverified_scene_numbering, last_search_time, episode_file_id (nullable).
- `episode_files` — series_id, season_number, relative_path, size, date_added, original_file_path, release_group, quality (JSONB), languages (JSONB), media_info (JSONB), indexer_flags, custom_format_score, custom_formats (JSONB), cf_fingerprint. One file can be referenced by multiple episodes (multi-episode releases).

**Profile / policy (grouped singleton config tables, not key-value)**
- `quality_profiles`, `custom_formats`, `release_profiles`, `delay_profiles`, `auto_tagging_rules`, `naming_config`, `media_management_config`, `indexer_config`, `ui_config`, `host_config`.
- `quality_definitions` — ~30 built-in quality rows, seeded at startup.

**Provider instances**
- `indexers`, `download_clients`, `notifications`, `import_lists`, `metadata_consumers`, `metadata_sources`.
- All share shape `(id, name, implementation, settings JSONB, enabled, priority, tags, added)` plus kind-specific flags.
- `*_status` tables per provider kind for failure backoff.
- `import_list_exclusions` — `(tvdb_id, title)`.

**Operational / state**
- `commands` — durable work queue with `worker_id`, `lease_until` for crash recovery.
- `scheduled_tasks` — `(type_name, interval_seconds, last_execution, next_execution)`.
- `queue` — active downloads.
- `history` — append-only event log, partitioned monthly on Postgres, trimmed by housekeeping on SQLite.
- `blocklist`, `pending_releases`, `tracked_downloads`.

**Supporting**
- `root_folders`, `remote_path_mappings`, `tags`, `users`, `sessions`, `metadata_files`, `subtitle_files`, `extra_files`, `scene_mappings`, `media_covers`, `log_entries` (separate from main schema), `series_statistics` (cached aggregates).

### 4.2 Dialect strategy

- Migrations under `internal/db/migrations/{postgres,sqlite}/NNN_name.sql`.
- `sqlc` generates typed Go code per dialect; dialect-neutral queries share a file, dialect-specific queries split.
- **Postgres**: JSONB fields, TIMESTAMPTZ, BIGSERIAL PKs, table partitioning for history.
- **SQLite**: TEXT for JSON fields, INTEGER unix-millis for timestamps, INTEGER PRIMARY KEY (rowid alias).
- Repository interfaces (`SeriesStore`, `EpisodeStore`, etc.) are dialect-agnostic; domain code depends only on interfaces. CI runs every integration test twice (once per dialect).

### 4.3 Key indexes

```
episodes (series_id, season_number, episode_number) UNIQUE
episodes (air_date_utc)
episodes (episode_file_id) WHERE episode_file_id IS NOT NULL
episode_files (series_id, season_number)
episode_files (cf_fingerprint)
commands (status, priority DESC, queued_at ASC)
commands (lease_until) WHERE status = 'running'
history (series_id, date DESC)
history (episode_id, date DESC)
history (date DESC)
queue (download_id) UNIQUE
pending_releases (series_id, added)
scheduled_tasks (next_execution)
series (path) UNIQUE
series (tvdb_id) UNIQUE
series (title_slug) UNIQUE
```

Full-text search on series uses Postgres `tsvector` GIN / SQLite FTS5 on `(title, overview, network, actors)`.

### 4.4 Connection pooling and write discipline

- **Postgres**: `pgxpool` with `max_conns=20`, `min_conns=2` defaults tuned for 2-core homelab targets.
- **SQLite**: WAL mode, `busy_timeout=5000`. All writes funnel through a single writer goroutine fed by a channel; reads use a separate read-only connection pool. Never wait on SQLite's native locking — we make the constraint explicit.
- **Never wrap external I/O in a DB transaction.** Metadata fetch, indexer call, download client call — all done outside transactions. DB transactions cover only the write, with pre-fetched data.

## 5. Execution Engine

### 5.1 Command queue

Every scheduled task, user action, and post-event follow-up is a row in `commands`. Sonarr's model, with three fixes:

**Crash-recovery leases**. Workers write `worker_id` and `lease_until = now() + 5 min` on claim, refreshing while running. The scheduler sweeps expired leases on startup and every 30s, re-queuing them. Eliminates "stuck running" commands.

**Deduplication**. On enqueue, a `dedup_key` is computed from `(name, canonical(body))`. Idempotent commands (e.g., `RssSync`) return the existing id; non-idempotent commands reject with a typed error.

**Priority + fairness**. Three priorities: `High` (user actions), `Normal` (scheduled tasks, post-download), `Low` (housekeeping, media covers). Workers pull highest-priority first; within priority, FIFO.

**Claim query (Postgres)**:
```sql
UPDATE commands
SET status='running', worker_id=$1, started_at=now(), lease_until=now() + interval '5 minutes'
WHERE id = (
  SELECT id FROM commands
  WHERE status='queued' AND (not_before IS NULL OR not_before <= now())
  ORDER BY priority DESC, queued_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
RETURNING *;
```

On SQLite, all claims funnel through the single writer goroutine; execution remains concurrent.

### 5.2 Scheduler

Single coordinator goroutine, ~500ms tick. Each tick:

1. Query `scheduled_tasks` for due rows.
2. For each, enqueue the command, update `last_execution` and `next_execution`.
3. Sweep expired leases.
4. Notify workers via a non-blocking signal channel.

**Scheduled task intervals**:

| Task | Interval | Notes |
|------|----------|-------|
| `RefreshMonitoredDownloads` | 1 min | Drives UI queue state. |
| `RssSync` | 15 min default, configurable ≥ 10 min | Parallel fetch across indexers. |
| `CheckHealth` | 1 h | |
| `RefreshSeries` | **Event-driven, not scheduled** | See §6.1. |
| `ImportListSync` | 6 h | Not 5 min like upstream. |
| `Backup` | Configurable: 1/7/30 days | |
| `Housekeeping` | 24 h | |
| `CleanUpRecycleBin` | 24 h | |
| `UpdateCheck` | 6 h | |
| `MessagingCleanup` | 1 h | Trims completed commands rows. |

All intervals are editable at runtime.

### 5.3 Worker pool

- Size: `min(NumCPU, 4)` default, 1-16 configurable.
- Each worker: `claim → dispatch → report → release → repeat`.
- Dispatch is a `map[string]Handler` populated at startup.
- Handlers receive a `context.Context` with cancellation, a `Progress` reporter (throttled to 1 update/sec per command), and a command-scoped logger.
- Panic recovery: marks command failed, fires event, does not bring down the process.
- **Exclusive commands** (e.g., `Housekeeping`, `Backup`) acquire a DB advisory lock on Postgres or an in-memory mutex on SQLite.

### 5.4 Event bus

In-process, typed, no external broker.

Because Go does not support generic methods on interfaces, the typed subscribe operations are expressed as package-level generic functions that operate on a non-generic `Bus` interface:

```go
type Bus interface {
    Publish(ctx context.Context, event any) error
    register(eventType reflect.Type, h handlerEntry)
}

func SubscribeSync[T any](bus Bus, handler func(ctx context.Context, e T) error) { ... }
func SubscribeAsync[T any](bus Bus, handler func(ctx context.Context, e T)) { ... }
func SubscribeOrdered[T any](bus Bus, order int, handler func(ctx context.Context, e T) error) { ... }
```

- **Sync handlers** block the publisher and propagate errors (e.g., `EpisodeImported` → `UpdateEpisodeFileRecord` must finish before `NotifyMediaServer` fires).
- **Async handlers** are fire-and-forget on a bounded goroutine pool (notifications, UI broadcasts, log shipping).
- **Ordered handlers** run in numeric order within the sync phase.
- Registration at DI wiring time, not runtime.
- Internally `map[reflect.Type][]handler`. Publish rate is well under 1000 events/sec even at peak.

**Event categories**: lifecycle (`AppStarted`, `AppStopping`, `ConfigChanged`); series (`SeriesAdded`, `SeriesRefreshed`, `SeriesEdited`, `SeriesDeleted`); episode (`EpisodeInfoRefreshed`, `EpisodeFileAdded`, `EpisodeFileDeleted`, `EpisodeFileRenamed`, `EpisodeFileMoved`); release (`ReleasesGrabbed`, `ReleaseFailed`, `ReleaseBlocked`); queue (`QueueItemAdded`, `QueueItemUpdated`, `QueueItemRemoved`); health (`HealthCheckCompleted`, `HealthIssueRaised`, `HealthIssueResolved`); command (`CommandQueued`, `CommandStarted`, `CommandCompleted`, `CommandFailed`, `CommandProgress`); media (`ManualInteractionRequired`, `ImportCompleted`, `DownloadFailed`).

### 5.5 Filesystem watcher

One `fsnotify` watcher per root folder, owned by a single goroutine:

1. Receives raw events.
2. Coalesces events within a 2-second window on the same path.
3. Resolves path → series_id via prefix lookup against a sorted `series.path` slice.
4. Enqueues a **targeted** `ScanSeriesFolder` for that series only.

This is the primary mechanism that breaks Sonarr's blanket 12-hour refresh behavior (see §6.1).

### 5.6 Graceful shutdown

On SIGTERM/SIGINT:

1. Close HTTP listener (in-flight requests drain).
2. Close scheduler tick channel.
3. Cancel the root context.
4. Wait up to 30s for in-flight commands to finish.
5. Leases on remaining commands expire naturally; they re-queue on restart.
6. Close DB pools, flush logs, exit.

## 6. Performance Strategy

This section enumerates each known upstream pain point and the specific architectural fix.

### 6.1 Kill the 12-hour blanket refresh

**Upstream problem**: `RefreshSeries` runs every 12h and does metadata fetch + disk walk + custom format recompute on every series. A 29k-series library touches all 29k folders twice a day.

**Fix**: split into three independent operations, each driven by the signal that matters.

1. **Metadata refresh** — staggered and lazy. Active series refreshed every ~7 days, staggered by `hash(series_id) % 7 days`. Ended series every ~30 days. Push-driven updates when the metadata source publishes a "recently updated" feed. Newly added series refreshed immediately.
2. **Disk scan** — event-driven via `fsnotify`. The only full walks are first startup, new root folder, and manual "rescan everything."
3. **Custom format recalc** — fingerprint-gated, see §6.3.

Net: in steady state on 29k series, the work that upstream does every 12h is essentially nothing unless something actually changed.

### 6.2 Database locking

**Upstream problem**: constant "database is locked" errors because long operations hold SQLite write locks while doing CPU work.

**Fix**:
- **Postgres-first design**: hot paths assume row-level locks and MVCC.
- **SQLite discipline**: single writer goroutine, separate read pool with `query_only=1`. WAL as a safety net, not primary strategy.
- **No long transactions**. `RefreshSeries` opens a transaction per series, not per run.
- **No external I/O in transactions**. Fetch first, write second.
- **No `COUNT(*)` in hot paths**. UI counters come from `series_statistics`.

### 6.3 Custom format memoization

**Upstream problem**: CF matches recomputed per render, per decision, per RSS cycle (issue #5298 open).

**Fix**:
- Each `CustomFormat` has `definition_hash`. A singleton `custom_format_version` increments on any CF change.
- `episode_files` stores `custom_format_score`, `custom_formats` JSONB, and `cf_fingerprint`.
- A low-priority `RecalculateCustomFormats` command visits files with stale fingerprints in batches of ~200, yields, continues.
- UI reads stored scores; never recomputes at read time.
- Release matching caches `(release_title, fingerprint) → score` in an LRU (~10k entries). Release titles repeat across RSS cycles.

### 6.4 MediaCover resilience

**Upstream problem**: broken poster URL loops retries inline inside refresh (#5228).

**Fix**:
- Dedicated bounded worker pool (2 goroutines), independent of the main queue.
- Each cover: `last_attempt_at`, `last_success_at`, `failure_count`, `next_retry_at`. Exponential backoff: 1m → 5m → 30m → 2h → 12h → 24h cap.
- ETag / Last-Modified honored.
- Image resize via stdlib `image/jpeg` + `golang.org/x/image`.
- A broken cover never blocks a refresh.

### 6.5 Memory discipline

**Upstream problem**: Docker memory grows unbounded (#7711).

**Fix**: bounded everything.

- Command `result` JSONB capped at 64 KB.
- Per-command log buffer: 1 MB tail-retained.
- Release title LRU: 10,000 entries.
- Parser regex cache: 5,000 entries.
- MediaCover in-memory cache: 256 MB LRU.
- RSS release cache: 30-min TTL, 20k entries.
- Log DB trimmed hourly, 90-day default retention.
- HTTP bodies streamed or capped at 10 MB. No unbounded `io.ReadAll`.
- `/debug/pprof/` behind a debug flag. `GOMEMLIMIT` documented.

### 6.6 Release result caching

**Upstream problem**: failed download handling re-searches indexers (#8358).

**Fix**: `recent_releases` cache table keyed on `(series_id, season, episode, content_hash)` with 30-min TTL. On download failure, pick the next-ranked cached candidate — no indexer hit.

### 6.7 Large-library query patterns

- **Series list**: paginated, max page 200. Cursor pagination (not offset) for stable deep pagination. Full-text search via tsvector GIN / FTS5.
- **Episodes for a series**: fetched per-season on demand.
- **Calendar**: always bounded by date range, using `episodes(air_date_utc)` index.
- **Activity queue**: paginated, most-recent-first.
- **Statistics cache**: `series_statistics` with `(series_id, episode_count, episode_file_count, size_on_disk, percent_of_episodes)`. Updated via triggers (Postgres) or event handlers (SQLite). UI counters never do runtime aggregates.

### 6.8 Daily-show handling

- `series_type='daily'` uses `(series_id, air_date)` as primary lookup.
- Bulk inserts via `COPY` on Postgres / batched prepared statements on SQLite. Target: 10k episodes in <10 seconds.
- Import pipeline parses dates from filenames, looks up by date not S/E.
- UI shows daily shows in a date-grouped view.

### 6.9 Shared HTTP client

One `httpclient.Client` with:
- Per-host stdlib connection pool
- Per-host rate limiter (`golang.org/x/time/rate`)
- Retry with exponential backoff + jitter on 5xx / network errors
- Circuit breaker per host
- Structured request logging with correlation IDs

Provider implementations call `client.Do(ctx, req)` and get all of this.

### 6.10 Startup behavior

Lazy. HTTP + UI up within 2-3 seconds of DB migrations completing. Background systems (scheduler, workers, filesystem watcher, metadata refresh) start in parallel. UI shows "warming up" indicators where data isn't ready.

### 6.11 Benchmark targets

Reference: 10k series / 500k episode files, 2 CPU cores / 4 GB RAM, Postgres.

| Scenario | Target | Upstream baseline |
|----------|--------|-------------------|
| Cold startup to UI responsive | ≤ 5 s | 30-60 s |
| Series list first page (100) | ≤ 200 ms | 2-10 s |
| Series detail (50 episodes) | ≤ 300 ms | 1-5 s |
| Calendar view (14 days) | ≤ 200 ms | 2-8 s |
| Full-library metadata refresh | ≤ 10 min | 4+ hours |
| RSS sync (20 indexers) | ≤ 60 s | 30-120 s |
| Idle RSS memory | ≤ 400 MB | 800 MB - 2 GB |
| Custom format recalc (500k files) | ≤ 15 min background | unbounded |
| "Database is locked" errors / day | 0 | multiple |
| Max CPU at idle | ≤ 2% | 10-30% |

CI benchmarks fail the build on > 20% regression.

## 7. Provider SDK

### 7.1 Provider kinds

| Kind | Upstream count | sonarr2 v1 target |
|------|---------------|-------------------|
| Indexer | 9 | 9 |
| Download client | ~18 | ~18 |
| Notification | 27 | 27 |
| Import list | 8 | 8 |
| Metadata consumer (nfo/artwork writers) | 4 | 4 |
| Metadata source | 1 (Skyhook) | ≥ 2 (TVDB direct + TVDB proxy), extensible |

### 7.2 Shared provider contract

```go
type Provider interface {
    Implementation() string
    DefaultName() string
    Settings() any
    Test(ctx context.Context) error
}
```

Each kind extends `Provider` with kind-specific methods (`FetchRss`, `Search`, `Add`, `OnGrab`, `Fetch`, `SeriesMetadata`, etc.).

### 7.3 Package-per-provider layout

```
internal/providers/indexer/
├── indexer.go           # interfaces, shared types
├── registry.go          # Register / Get / All
├── newznab/
├── torznab/
├── broadcasthenet/
└── ... (9 total)
```

Zero cross-provider dependencies. Flat dependency graph. Plugin-ready without being a plugin.

### 7.4 Settings as typed structs with form metadata

```go
type Settings struct {
    BaseURL    string `json:"baseUrl"    form:"text"        label:"URL"         required:"true"`
    APIKey     string `json:"apiKey"     form:"password"    label:"API Key"     required:"true"`
    Categories []int  `json:"categories" form:"multiselect" label:"Categories"`
}
```

A reflection-based `providers.SchemaFor(settings any) Schema` walks struct tags and emits a JSON schema matching Sonarr's `ClientSchema` shape. The frontend renders forms dynamically via a shared `<ProviderForm>` component.

### 7.5 Metadata source — the Skyhook replacement

sonarr2 defines an abstract `MetadataSource` interface with:

```go
type MetadataSource interface {
    Provider
    GetSeries(ctx, externalID ExternalID) (Series, []Episode, error)
    SearchSeries(ctx, query string) ([]Series, error)
    SupportedExternalIDs() []ExternalIDKind
}
```

**Ships in v1**: two implementations.
- `tvdb_direct` — calls TheTVDB v4 API directly with a project API key (or user-supplied subscriber PIN).
- `tvdb_proxy` — calls a user-hostable `sonarr2-metadata-proxy` companion service (also shipped in v1) that fronts TVDB with caching, rate limiting, and response normalization.

**Future** (post-v1): `tmdb_direct`, `tmdb_proxy`, `tvmaze_direct`, `tvmaze_proxy`, and any other source. The `external_ids` table tracks cross-source IDs so users can switch primary sources without re-adding series.

### 7.6 Provider registration

At startup, providers register via `init()` in their package:

```go
// internal/providers/indexer/newznab/register.go
func init() {
    indexer.Register("Newznab", func() indexer.Indexer { return New() })
}
```

Main wires in providers via blank imports:
```go
import (
    _ "sonarr2/internal/providers/indexer/newznab"
    _ "sonarr2/internal/providers/downloadclient/sabnzbd"
    // ...
)
```

`init()` is used here intentionally; everything else uses explicit DI.

### 7.7 No runtime plugin loading in v1

Providers are compiled in. Reasoning: Go's `plugin` package is fragile, out-of-process plugins add complexity, community contributions happen via PR. Interfaces are designed to support out-of-process plugins later via gRPC or WASM without touching provider code.

### 7.8 Status tracking (decorator pattern)

Each kind has a `StatusService`; instances are wrapped with a decorator on creation:

```go
type statusTrackingIndexer struct {
    inner  Indexer
    id     int64
    status StatusService
}
```

Failures update per-instance status with exponential backoff. Disabled instances return a `disabledIndexer` that errors immediately.

### 7.9 Provider testing

Each package ships tests against recorded fixture data (VCR-style). No real external calls in CI. Live integration tests behind a build tag run nightly with secrets.

## 8. Decision Engine and Import Pipeline

### 8.1 Parsing (pure, testable)

**Title parser** (`internal/parser/title.go`) — pure function, LRU cache of 5000 entries. Three variants tried in order: standard (SxxExx), daily (YYYY.MM.DD), anime (absolute + hash).

**Quality parser** (`internal/parser/quality.go`) — pure function. Detects source, resolution, modifier, version. Maps to a `quality_definitions` row.

**Series matching** (`internal/parser/seriesmatch.go`) — not pure (touches DB via injected `SeriesLookup`). Normalized lookup, fuzzy fallback with tight thresholds.

Golden-file tests via `testdata/titles.json` with hundreds of known-good and known-problem titles. Every parser bug report adds a row.

### 8.2 Custom format matching

Spec kinds mirror upstream exactly (so TRaSH Guides JSON imports work):

- `ReleaseTitle` (regex)
- `ReleaseGroup` (regex)
- `ReleaseType` (single / season pack / multi-season)
- `Source` / `Resolution` / `Language`
- `IndexerFlag` (freeleech / internal / scene / etc.)
- `Size` (min/max bytes)
- Negated variants for all of the above

Scoring and memoization per §6.3.

### 8.3 Decision engine

Each release runs through ~30 specifications collected in `internal/decisionengine/specs/`, one file per spec:

Blocklist, BlockedIndexer, AcceptableSize, MaximumSize, AirDate, AlreadyImported, MinimumAge, Retention, Protocol, FreeSpace, QualityAllowedByProfile, LanguageAllowedByProfile, CustomFormatAllowedByProfile, Queue, NotSample, SeasonPackOnly, SameEpisodes, Upgradable, UpgradeAllowed, UpgradeDisk, RawDisk, Repack, FullSeason, MultiSeason, SplitEpisode, AnimeVersionUpgrade, TorrentSeeding, SceneMapping, ReleaseRestrictions.

Each returns `Accept` or `Reject(reasons)`. The engine collects all rejections for the UI's "Why wasn't this grabbed?" panel.

### 8.4 Release ranking

Surviving releases ordered by:

1. Custom format score (higher better)
2. Quality (per profile order)
3. Language preference
4. Protocol preference (per delay profile)
5. Indexer priority (lower number = preferred)
6. Size proximity to preferred
7. Seeders (torrent)
8. Age (usenet: older; torrent: newer)
9. Lexicographic title (deterministic tiebreaker)

### 8.5 Grab flow

```
Indexer RSS / Manual Search
       │
       ▼
 Parse → Match → Decision Engine → Rank
       │
       ▼
 Delay profile check ── holds? → pending_releases
       │                           │
       ▼                           ▼
 Pick top             CheckForPendingReleases (60s)
       │                           │
       ▼◄──────────────────────────┘
 Download client send
       │
       ▼
 [Transaction: queue row + history row]
       │  (on DB failure → compensate: remove from download client)
       ▼
 Fire ReleasesGrabbed event → notifications
```

The grab transaction inverts upstream's pattern: DB write is the source of truth; download client call happens first and is compensated on DB failure.

### 8.6 Delay profiles

Fields: `EnableUsenet`, `EnableTorrent`, `PreferredProtocol`, `UsenetDelay`, `TorrentDelay`, `BypassIfHighestQuality`, `BypassIfAboveCustomFormatScore`, `MinimumCustomFormatScore`, `Tags`.

Scheduled `CheckForPendingReleases` task every 60s grabs expired entries, re-runs the decision engine, grabs the winner.

### 8.7 Import pipeline

Triggered by download client poll, filesystem watcher on download dir, or manual import.

```
Downloaded Item Discovered
       │
       ▼
 Locate files in download dir
       │
       ▼
 For each candidate file:
   parse → match series → match episodes → import specs
       │
       ├─ Rejected → held for manual import
       │
       └─ Accepted:
           Determine destination path (naming tokens + series.path)
           Hardlink? Copy? Move?
           ffprobe for MediaInfo
           Write episode_files + update episodes (transaction)
           Fire EpisodeImported event → notifications, metadata consumers, media server refresh
           History: Downloaded event
           Optional: tell download client to remove
```

**Import specifications** (`internal/import/specs/`):

NotSample, NotUnpacking, AlreadyImported, FreeSpace, MatchesFolder, MatchesGrab, HasAudioTrack, UnverifiedSceneNumbering, FullSeason, SplitEpisode, AbsoluteEpisodeNumber, EpisodeTitle, Upgrade.

### 8.8 File operations

- **Hardlink-first**: same-filesystem detection via `syscall.Stat_t.Dev` → hardlink. Instant, zero disk cost, preserves seeding.
- **Copy + delete** otherwise: `io.Copy` with 1 MB buffer, progress events.
- **Move as fallback**: `os.Rename` if configured.
- **Atomic visibility**: write to `.partial`, rename to final name.
- **Permissions**: chmod/chown from config, fallback to parent inherit.
- **Recycle bin**: timestamped move, not delete. Housekeeping trims old entries.
- **Extras**: subtitle/nfo/artwork carried over and renamed to match.

### 8.9 Manual import

- `GET /api/v3/manualimport?folder=...` returns files + suggested parse + rejections.
- User adjusts, POSTs `{path, seriesId, episodeIds, quality, language, ...}` tuples.
- Import pipeline skips most specs (user explicitly chose), still runs `FreeSpace`, `NotUnpacking`, destination validation.

### 8.10 Scene mappings

`scene_mappings` table refreshed from TheXEM API for anime, plus a small in-repo JSON for non-anime corrections. Not dependent on Servarr-hosted data.

## 9. API and Transport Layer

### 9.1 Two API versions, side by side

- **`/api/v3`** — wire-compatible with Sonarr's current stable API. Every endpoint, field name, status code, and pagination shape matches. This is the drop-in commitment.
- **`/api/v6`** — fresh surface: cursor pagination, RFC 9457 error envelope, typed event streams, ETags, consistent naming.

Both mount on the same underlying domain services. v3 handlers translate to/from Sonarr's exact field names; v6 handlers use sonarr2's shapes. Duplicate code is acceptable to isolate wire-format concerns from domain logic.

**v3 ground truth**: Sonarr's `openapi.json`. CI includes a schema-diff job against a live upstream Sonarr test instance.

### 9.2 HTTP framework

`net/http` + `go-chi/chi/v5` router. No framework magic. Handlers accept `http.ResponseWriter, *http.Request`. Thin `render` package for JSON.

### 9.3 Authentication

Four modes (configurable in `host_config`), matching Sonarr:

1. **None** — dev/LAN only, with warning banner.
2. **Basic** — HTTP Basic for reverse-proxy scenarios.
3. **Forms** — session cookie, login page at `/login`, `HttpOnly`/`SameSite=Lax`/`Secure` cookies.
4. **External** — trust a configured header (e.g., `X-Forwarded-User`) only from trusted proxy CIDRs.

**API key** as a separate channel on top of any of the above. `X-Api-Key` header or `?apikey=` query. Stored in `host_config.api_key`, generated on first run, preserved during migration from upstream Sonarr so existing integrations keep working.

Authorization: single-user for v1.

Middleware chain: `log → recover → cors → auth → rateLimit → handler`.

### 9.4 Request/response conventions

- Content type: `application/json; charset=utf-8`. No form-encoded POSTs, no XML.
- **v3 pagination** (matches Sonarr):
  ```json
  {"page": 1, "pageSize": 50, "sortKey": "sortTitle", "sortDirection": "ascending", "totalRecords": 1482, "records": [...]}
  ```
- **v6 pagination** (cursor):
  ```json
  {"data": [...], "pagination": {"limit": 50, "nextCursor": "...", "hasMore": true}}
  ```
- **v3 error envelope** (matches Sonarr):
  ```json
  {"message": "Series not found", "description": "No series with id 42"}
  ```
  Validation:
  ```json
  [{"propertyName": "title", "errorMessage": "Required"}]
  ```
- **v6 error envelope** (RFC 9457):
  ```json
  {"type": "...", "title": "Series not found", "status": 404, "detail": "...", "instance": "/api/v6/series/42"}
  ```
- Validation: `go-playground/validator` wrapped in a helper that emits the right envelope for the mount point.

### 9.5 CORS

- v3: permissive for API key auth, same-origin only for cookie auth (matches Sonarr).
- v6: configurable allowlist via `host_config.allowed_origins`, defaults to same-origin.

### 9.6 Rate limiting

Per-IP limiter on `/api/*` via `golang.org/x/time/rate` + LRU. Generous defaults (100 req/sec burst, 30 sustained). DoS protection, not throttling. API-key auth bypasses per-IP limits.

### 9.7 OpenAPI

- v3: hand-maintained `internal/api/v3/openapi.json` matching Sonarr field-for-field. Served at `/api/v3/openapi.json`.
- v6: generated from Go code or hand-maintained. Swagger UI at `/docs/api/v6`.

### 9.8 Real-time transport

**Two transports, side by side, both publishing the same event stream.**

**SignalR emulation at `/signalr/messages`** — enough of the SignalR protocol to make existing Sonarr clients work. Negotiation, WebSocket upgrade, message envelopes matching Sonarr's shapes. Library candidate: `philippseith/signalr` (pending maintenance check); fallback is hand-implementing the needed subset.

**Server-Sent Events at `/api/v6/stream`** — standard SSE with event types, JSON payloads, `?events=` filter, `Last-Event-ID` resume from a ring buffer (last 1000 events).

**`realtime.Broker`** subscribes to the internal event bus and maps domain events to broadcast events. Broadcasts are fire-and-forget with bounded per-client buffers. Slow clients are dropped with a warning log, never blocking domain operations.

**Event types broadcast** (matching Sonarr's `SignalRMessage` shapes):
series, episode, episodefile, queue, history, wanted/missing, wanted/cutoff, command, health, log, tag, rootfolder, indexer, downloadclient, notification, calendar.

### 9.9 Static asset serving

- Frontend embedded via `//go:embed frontend_dist/*`.
- `/` serves `index.html` (no-cache). `/assets/*` serves hashed bundles (immutable, long cache).
- `/mediacover/{seriesId}/{filename}` serves on-disk cover files.
- SPA fallback: unknown paths under `/` return `index.html`.

### 9.10 API testing

- Per-handler unit tests with `httptest.NewRecorder`.
- Per-version integration tests with `httptest.NewServer` + in-memory DB.
- **v3 wire-compat tests**: golden response files captured from real Sonarr, diffed with configurable exclusions.
- Load tests via `vegeta`/`ghz` against the reference dataset. Gate merges on p95 latency targets per §6.11.

## 10. Frontend Architecture

### 10.1 Stack

| Layer | Choice |
|-------|--------|
| Build | Vite + `@vitejs/plugin-react` |
| Language | TypeScript strict mode (`strict: true`, `noUncheckedIndexedAccess: true`) |
| UI | React 18 |
| Routing | React Router v6 (data router API) |
| Server state | TanStack Query v5 |
| Local UI state | Zustand |
| Forms | React Hook Form + Zod |
| Tables | TanStack Table v8 (headless) |
| Virtualization | TanStack Virtual |
| Drag-and-drop | dnd-kit |
| Styling | CSS Modules + CSS custom properties (design tokens) |
| Interactive primitives | Radix UI |
| Icons | Lucide React |
| Dates | date-fns |
| Charts | Recharts |
| i18n | i18next + react-i18next |
| Testing | Vitest (unit), Testing Library (component), Playwright (e2e) |

Redux is intentionally **not** used. TanStack Query handles server state; Zustand handles local UI state. This is a meaningful cleanup over upstream Sonarr.

### 10.2 Styling: CSS Modules + design tokens

Design tokens in `frontend/src/styles/tokens.css` via CSS custom properties. Dark theme default, light theme as secondary.

Example tokens: `--color-background`, `--color-surface`, `--color-accent`, `--space-1..8`, `--radius-*`, `--font-sans`, `--font-mono`. Values sampled from running-Sonarr screenshots, not copied from Sonarr's SCSS.

Radix UI primitives (Dialog, Popover, DropdownMenu, Tabs, Tooltip, Select, Toast, Switch) are unstyled by default and styled via CSS Modules.

### 10.3 Directory layout

```
frontend/src/
├── main.tsx
├── App.tsx
├── api/
│   ├── client.ts
│   ├── series.ts               # useSeries, useSeriesList, useUpdateSeries, ...
│   ├── episodes.ts
│   ├── queue.ts
│   ├── commands.ts
│   ├── types.ts
│   └── generated/              # openapi-typescript output (committed)
│       ├── v3.ts
│       └── v6.ts
├── components/                 # shared primitives on top of Radix
├── layout/                     # AppShell, Sidebar, TopBar, PageHeader
├── pages/
│   ├── Series/
│   ├── Calendar/
│   ├── Activity/               # Queue, History, Blocklist (tabs)
│   ├── Wanted/                 # Missing, CutoffUnmet
│   ├── Settings/
│   │   ├── MediaManagement/
│   │   ├── Profiles/
│   │   ├── Quality/
│   │   ├── CustomFormats/
│   │   ├── Indexers/
│   │   ├── DownloadClients/
│   │   ├── ImportLists/
│   │   ├── Connect/
│   │   ├── Metadata/
│   │   ├── Tags/
│   │   ├── General/
│   │   └── UI/
│   ├── System/
│   │   ├── Status/
│   │   ├── Tasks/
│   │   ├── Backup/
│   │   ├── Updates/
│   │   ├── Events/
│   │   └── Logs/
│   └── Login/
├── providers/                  # QueryProvider, RealtimeProvider, ThemeProvider, ToastProvider
├── hooks/                      # useRealtimeEvent, useDebouncedValue, usePagination
├── features/
│   ├── providerForm/           # dynamic provider settings forms
│   ├── interactiveSearch/
│   ├── manualImport/
│   ├── filenamePreview/
│   └── seasonPass/
├── i18n/
│   ├── config.ts
│   └── locales/
├── styles/
└── utils/
```

### 10.4 Data fetching

All API calls go through typed TanStack Query hooks. No direct `fetch` from components.

```typescript
export function useSeries(id: number) {
  return useQuery({
    queryKey: ['series', id],
    queryFn: () => apiClient.get<Series>(`/api/v3/series/${id}`),
    staleTime: 60_000,
  });
}
```

**Realtime-driven invalidation**: `RealtimeProvider` subscribes to the SSE/SignalR stream and invalidates React Query keys on matching events. Cache + real-time updates cleanly separated.

### 10.5 Dynamic provider forms

`<ProviderForm schema={...} values={...} onChange={...} />` fetches the schema from the backend and renders fields dynamically. Field type map: text / password / number / checkbox / select / multiselect / tag / path. Any new backend provider works with zero frontend changes.

### 10.6 Virtualization

`SeriesIndexPage` and long episode tables (Jeopardy-class) use TanStack Virtual. The backend paginates, but within a loaded page we virtualize to protect against huge `pageSize` requests.

### 10.7 Type sharing

TypeScript types generated from OpenAPI:

- `frontend/src/api/generated/v3.ts` — from `/api/v3/openapi.json`
- `frontend/src/api/generated/v6.ts` — from `/api/v6/openapi.json`

Generator: `openapi-typescript`. Runs in CI. Drift between backend schema and committed types fails the build. End-to-end type safety.

### 10.8 Theming and accessibility

- Dark theme default via `data-theme="dark"` on `<html>`.
- Radix primitives handle focus, ARIA, and keyboard navigation.
- axe-core in CI checks WCAG AA contrast.
- Full keyboard navigation required for every action.

### 10.9 Internationalization

`i18next` with JSON locale files. English is source; other languages contributed. Date/time/number formatting via `Intl.*`.

### 10.10 Dev workflow

- `npm run dev` — Vite on `:5173` with proxy to `:8989`.
- `npm run build` — production build to `frontend/dist/`, embedded into the Go binary.
- `npm run generate:types` — regenerate OpenAPI-derived types.
- Single binary, no separate frontend server in production.

### 10.11 Testing

- **Vitest** unit tests for utilities, hooks, logic.
- **Testing Library** component tests.
- **Playwright** e2e for critical flows: login, add series, manual search, grab, import.
- **Storybook** for the design system.
- **Visual regression** via Playwright screenshot diffs for the Storybook components and key page layouts. Baselines committed; CI fails on unapproved pixel diffs.

### 10.12 Bundle budget

- Initial JS < 500 KB gzipped.
- Route-based code splitting via `React.lazy`.
- Tree-shaken icons and date-fns.
- No moment.js, no lodash, no jQuery.
- Bundle analyzer in CI with 10% regression threshold.

## 11. Migration Tool

Standalone `sonarr-migrate` binary under `tools/sonarr-migrate/`. Reads an existing Sonarr database (v4.x or v5.x), populates a fresh sonarr2 database.

### 11.1 Scope

**In scope**: series/seasons/episodes/episode_files records, all profiles and config, all provider definitions with settings, tags, root folders, remote path mappings, history, blocklist, import list exclusions, users, host config (API key preserved).

**Out of scope**: command queue, active download queue, media covers (re-downloaded), log entries, scheduled task state, statistics (recomputed).

### 11.2 Supported source versions

v1 of the migrator supports Sonarr v4.0.0 through current v5.x. Detected from the `VersionInfo` table. Outside the window → refuse with an upgrade-Sonarr-first error.

### 11.3 Source reader abstraction

```go
type Reader interface {
    Version() string
    Dialect() Dialect // sqlite | postgres

    Series(ctx) (<-chan SourceSeries, error)
    Episodes(ctx) (<-chan SourceEpisode, error)
    EpisodeFiles(ctx) (<-chan SourceEpisodeFile, error)
    QualityProfiles(ctx) ([]SourceQualityProfile, error)
    CustomFormats(ctx) ([]SourceCustomFormat, error)
    // ... all other entity types
    Close() error
}
```

Per-version implementation: `tools/sonarr-migrate/reader/v4/`, `reader/v5/`. Strictly read-only source access (`?mode=ro&immutable=1` for SQLite; read-only transaction for Postgres).

### 11.4 Transformer layer

Pure functions per entity. `TransformContext` carries lookup tables (`sourceQualityProfileID → destQualityProfileID`) built in earlier passes, plus the path remapping function. Transformers emit `Issue` values for user-facing warnings. Table-driven unit tests per transformer against fixture data.

### 11.5 Writer layer

Writes go through the main app's repository layer (`import "sonarr2/internal/db"`), enforcing every domain invariant. Batched transactions of 500-1000 rows. Progress bar per entity group.

### 11.6 Migration order

Topologically ordered: `host_config → root_folders → remote_path_mappings → tags → quality_definitions → quality_profiles → custom_formats → release_profiles → delay_profiles → naming_config → media_management_config → indexer_config → indexers → download_clients → import_lists → notifications → metadata_consumers → import_list_exclusions → auto_tagging_rules → series → episodes → episode_files → blocklist → history → users`.

Each step records in a `_migration_state` table. On mid-run failure, `--resume` picks up from the failed entity. The main app refuses to start if `_migration_state` has non-completed rows.

### 11.7 Dry run mode

`--dry-run` runs every transformer and validator, writes nothing, emits a summary report and detailed `migration_report.json`. Users iterate on source data or migrator config before committing.

### 11.8 Path remapping

```
--remap /tv:/media/tv --remap /anime:/media/anime
```

Prefix-based, longest match wins. `--remap-all-or-refuse` fails if any path isn't remapped, ensuring no volume is missed.

### 11.9 Provider compatibility matrix

Per-kind implementation name map. Unsupported implementations are logged, skipped, included in the issues report. Settings fields are explicitly mapped per-implementation.

### 11.10 Validation pass

Post-write: row count comparison, 100-random-series episode/file spot-check, FK integrity, provider settings deserialization check. Warnings by default, fatal with `--strict`.

### 11.11 CLI

```
sonarr-migrate [flags]

Source:
  --source-type {sqlite|postgres}
  --source-path <file>              (sqlite)
  --source-dsn <dsn>                (postgres)
  --source-config <sonarr config.xml>

Destination:
  --dest-type {sqlite|postgres}
  --dest-path <file>                (sqlite)
  --dest-dsn <dsn>                  (postgres)

Behavior:
  --dry-run                         validate without writing
  --resume                          resume from last incomplete entity
  --skip-history
  --history-since <duration>
  --remap <src:dst>                 repeatable
  --remap-all-or-refuse
  --strict
  --into-existing
  --force
  --verbose
  --report <path>
```

### 11.12 Safety rails

- Source read-only.
- Destination empty unless `--into-existing`.
- Transactional per entity group.
- `--rollback` deletes migrator-written rows.
- Pre-flight: free disk ≥ 2× source data.
- Source `config.xml` read for API key preservation.

### 11.13 Testing

- Unit tests per transformer against fixtures.
- Integration test: migrate a real (anonymized) Sonarr SQLite DB, boot the main app, verify access.
- `--dry-run` CI against curated fixtures per source version.

## 12. Operations

### 12.1 Testing strategy (app-wide)

- **Unit** (~70%): pure functions, parsers, decision specs, transformers. 95%+ line coverage on `internal/parser/`, `internal/decisionengine/specs/`, `internal/customformats/`, `internal/organizer/`, `internal/import/specs/`.
- **Integration** (~25%): in-memory SQLite + `testcontainers-go` Postgres, real HTTP via `httptest.NewServer`, stubbed external HTTP.
- **E2E** (~5%): Playwright against real backend + Postgres. Critical journeys: login → add series → manual search → grab → queue update → import complete.
- **Benchmarks**: `go test -bench` per hot-path package, `benchstat` for comparison, >20% regression fails the build.
- **Property-based**: `gopter` for parser round-trip, quality comparator transitivity, decision engine idempotence.
- **Test database matrix**: every integration test runs against SQLite and Postgres.

### 12.2 Logging

Structured via `log/slog` (stdlib). JSON default, `--log-format text` for local dev. Every entry: `level, time, msg, component, correlation_id, series_id/episode_id/download_id` where relevant.

Routing:
- **stderr**: everything at configured level
- **`log_entries` table**: WARN and ERROR for UI log viewer, 90-day retention
- **Rotating files**: default 7 × 10 MB under `<config-dir>/logs/`

### 12.3 Metrics

Behind a config flag. `/metrics` exposes Prometheus metrics:
`sonarr2_command_queue_depth`, `sonarr2_command_duration_seconds_bucket`, `sonarr2_indexer_request_duration_seconds_bucket`, `sonarr2_indexer_failures_total`, `sonarr2_download_client_items`, `sonarr2_series_total`, `sonarr2_episode_files_total`, `sonarr2_database_query_duration_seconds_bucket`, `sonarr2_http_requests_total`, `sonarr2_http_request_duration_seconds_bucket`, `sonarr2_goroutines`, `sonarr2_memory_bytes`, `sonarr2_gc_pause_seconds`.

Library: `prometheus/client_golang`. Grafana dashboard JSON in `contrib/grafana/`.

### 12.4 Tracing

OpenTelemetry behind a config flag. OTLP export. Spans around HTTP, commands, decision engine, import pipeline, provider calls. Zero overhead when disabled.

### 12.5 Health checks

`internal/health/checks/`, fired on startup, periodically, and event-driven. Full parity with upstream Sonarr: AppDataLocation, ApiKeyValidation, Authentication, DownloadClient, DownloadClientStatus, DownloadClientSorting, DownloadClientRootFolder, DownloadClientRemovesCompletedDownloads, ImportList, ImportListStatus, ImportListRootFolder, Indexer, IndexerStatus, IndexerRss, IndexerSearch, IndexerLongTermStatus, IndexerDownloadClient, IndexerJackettAll, NotificationStatus, Proxy, RootFolder, RecycleBin, RemotePathMapping, RemovedSeries, SystemTime, Mount, Update, PackageGlobalMessage, ImportMechanism.

### 12.6 Housekeeping

`internal/housekeeping/`, daily at 3 AM local:

- Trim log entries past retention
- Trim completed commands past retention
- Delete orphan episode_file rows
- Delete orphan blocklist entries for removed series
- Delete orphan history entries
- Delete orphan metadata/subtitle/extra file rows
- Clean the recycle bin
- Compact DB (`VACUUM` on SQLite, no-op on Postgres)
- Delete unused media cover files
- Fix future-timestamp rows
- Recalculate `series_statistics` for stale rows

### 12.7 Backup and restore

**Backup**: `BackupCommand` (scheduled + manual). Tar.gz of DB + config.json + manifest. Excludes log DB and media covers (regenerable). Retention configurable (default 7 backups).

**Restore**: from UI or CLI. Refuses while server is running. Validates manifest version; runs forward migrations if source is older. Temp-DB → validate → atomic swap.

### 12.8 Update mechanism

- **Default**: none. Docker users update by pulling.
- **Optional built-in updater** for bare-binary users: checks GitHub Releases, UI banner, downloads + signature verifies (cosign) + swaps on restart. Opt-in via `host_config.update_mechanism`.

### 12.9 Build

```bash
make build         # frontend + backend → ./dist/sonarr2
make test
make lint          # gofmt, govet, staticcheck, golangci-lint, eslint, tsc
make docker
make docker-multi  # multi-arch (CI)
```

### 12.10 CI (GitHub Actions)

- `lint.yml` — Go + TS lint, fast (<2 min)
- `test.yml` — unit + integration with SQLite and Postgres matrix (~5-10 min)
- `e2e.yml` — Playwright (~10-15 min)
- `bench.yml` — benchmark suite with `benchstat` diff, PR comment
- `api-compat.yml` — diff v3 endpoints against a real Sonarr container
- `build-release.yml` — triggered on tags, multi-arch binaries + Docker images, cosign, GitHub release, GHCR + Docker Hub publish

### 12.11 Docker

```dockerfile
FROM node:20-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /src/frontend/dist /src/frontend/dist
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w -X sonarr2/internal/buildinfo.Version=$(git describe --tags --always)" \
    -o /out/sonarr2 ./cmd/sonarr

FROM gcr.io/distroless/static-debian12
COPY --from=backend /out/sonarr2 /sonarr2
EXPOSE 8989
VOLUME ["/config", "/media"]
ENTRYPOINT ["/sonarr2"]
CMD ["--config", "/config"]
```

`CGO_ENABLED=0` + pure-Go SQLite driver → no CGo toolchain friction, static binaries, simpler multi-arch.

### 12.12 Configuration

Precedence: CLI flags > env vars > config file > defaults.

Two tiers:

- **Host config** — immutable at runtime (port, bind, auth mode, paths, DB connection, API key). From file/env/flags at startup.
- **App config** — runtime-mutable via UI. Stored in DB.

### 12.13 Versioning

SemVer on the v3 API. Breaking v3 changes = major version. v6 starts at v0.x until stable. Release channels: `stable`, `nightly`. Docker tags: `:stable`, `:nightly`, `:vX.Y.Z`, `:vX.Y`, `:vX`.

### 12.14 Contributor hygiene

Root of repo: `README.md`, `LICENSE` (MIT), `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (Contributor Covenant), `SECURITY.md`, `.github/ISSUE_TEMPLATE/`, `.github/PULL_REQUEST_TEMPLATE.md`. Per-provider `README.md` inside each package.

## Appendix A — Provider inventory (full parity target)

### Indexers (9)

Newznab, Torznab, BroadcastheNet, FileList, HDBits, IPTorrents, Nyaa, Torrentleech, Fanzub.

### Download clients (~18)

SABnzbd, NZBGet, NzbVortex, Blackhole (usenet + torrent), Pneumatic, qBittorrent, rTorrent, uTorrent, Transmission, Deluge, Vuze, Flood, Aria2, Hadouken, FreeboxDownload, DownloadStation (Synology), Tribler, RQBit.

### Notifications (27)

Apprise, CustomScript, Discord, Email, Gotify, Join, Mailgun, MediaBrowser (Emby/Jellyfin), Notifiarr, Ntfy, Plex, Prowl, PushBullet, Pushcut, Pushover, SendGrid, Signal, Simplepush, Slack, Synology, Telegram, Trakt, Twitter, Webhook, Xbmc (Kodi).

### Import lists (8)

AniList, Custom (generic HTTP), MyAnimeList, Plex Watchlist, Rss, Simkl, Sonarr (other instance), Trakt (List / Popular / User).

### Metadata consumers (4)

Plex, Roksbox, Wdtv, Xbmc (Kodi).

### Metadata sources (v1: 2, extensible)

tvdb_direct, tvdb_proxy (sonarr2-metadata-proxy companion service). Future: tmdb_direct, tmdb_proxy, tvmaze_direct, tvmaze_proxy.

## Appendix B — Open questions

None blocking implementation. Items flagged during design for later revisit:

1. **SignalR Go library maintenance status** — verify `philippseith/signalr` is current before committing; hand-roll the subset if not.
2. **TVDB API key strategy** — negotiate an open-source project key vs. require user-supplied PIN. Decision deferred to the metadata source implementation phase.
3. **Scene mappings source** — depend on TheXEM (live API) vs. mirror upstream Sonarr's scene_mappings.json in our own repo. Decision deferred; both are viable.
4. **Community translation seeding** — verify provenance of Sonarr's public locale files before importing keys. If in doubt, start from English-only and seed other languages from community contributions.

## Appendix C — Glossary

- **CF** — Custom Format
- **DI** — Dependency Injection
- **FTS** — Full-Text Search
- **MVCC** — Multi-Version Concurrency Control (Postgres's concurrency model)
- **PVR** — Personal Video Recorder
- **SPA** — Single-Page Application
- **SSE** — Server-Sent Events
- **TVDB** — TheTVDB (TV metadata source)
- **WAL** — Write-Ahead Logging (SQLite journal mode)
