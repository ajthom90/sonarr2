# sonarr2

A feature-complete rewrite of [Sonarr](https://github.com/Sonarr/Sonarr) focused on performance for large TV libraries.

## Current Status

**M0–M23 complete; M24 (Sonarr parity) in progress.** All original roadmap
milestones shipped. M24 is a drop-in-replacement push aiming for byte-identical
v3 API compatibility and feature parity with upstream Sonarr (see
[M24 design doc](./docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md)).
Current M24 scope: navigation shell matches Sonarr's 6-item sidebar with all
sub-items; Tags / Blocklist / Remote Path Mappings / Recycle Bin / Release
Profiles / Delay Profiles / iCalendar feed landed end-to-end; provider
registries expanded to 10/10 indexers, 20/20 download clients, 24/25
notifications. Subsystems still pending: Import Lists, Metadata Consumers,
Auto-Tagging, Interactive Search, Manual Import, Scene Mappings, Custom Format
spec types beyond regex, expanded health checks, and additional scheduled
tasks. Not ready for end users (pre-v1).

### What's implemented

- **Database layer** — Postgres (pgx/v5) and SQLite (pure-Go, no CGo) with single-writer discipline preventing "database is locked" errors
- **Domain model** — Series, Seasons, Episodes, EpisodeFiles with dual-dialect stores and reactive statistics via a typed event bus
- **Command queue** — durable DB-backed queue with crash-recovery leases, deduplication, priority levels, and a worker pool with panic recovery
- **Scheduler** — tick-based background task scheduler with configurable intervals
- **Release parser** — extracts series title, season/episode numbers, quality (source + resolution + modifier), release group from release title strings; supports standard (SxxExx), daily (YYYY.MM.DD), and anime (absolute numbering) formats
- **Quality profiles** — 18 seeded quality definitions, configurable profiles with upgrade/cutoff logic
- **Custom formats** — regex-based format matching with weighted scoring (compatible with TRaSH Guides JSON)
- **Decision engine** — evaluates releases against quality profiles with 8 core specs; ranks accepted releases by CF score, quality, and size
- **Provider SDK** — pluggable integration architecture with reflection-based settings schema generation
- **Indexers (6)** — Newznab, Torznab, TorrentRss (full implementations); IPTorrents, Nyaa, BroadcastheNet (stubs)
- **Download clients (6)** — SABnzbd, NZBGet, qBittorrent, Transmission, Deluge, Blackhole
- **Notification providers (8)** — Discord, Slack, Telegram, Email, Webhook, Pushover, Gotify, CustomScript; event-driven dispatch on grab/download/health events
- **TVDB caching & rate limiting** — in-process TTL cache (24h series, 6h episodes, 1h search) with automatic invalidation on refresh; token-bucket rate limiter (5 req/s) with 429-aware exponential backoff
- **Health checks** — framework with 5 checks (database, root folders, indexers, download clients, metadata source); runs on startup and every 30 minutes; dispatches notifications for new issues
- **Housekeeping** — daily automated cleanup: history trimming (90-day configurable retention), orphan episode file removal, series statistics recalculation, SQLite VACUUM compaction
- **Automated backups** — tar.gz archives of SQLite database with JSON manifest; configurable retention (default 7) and schedule (default weekly); v3/v6 API for list/create/download/delete
- **Migration tool** — `sonarr-migrate` CLI imports series, episodes, episode files, quality profiles, indexers, download clients, and notifications from an existing Sonarr v3/v4 SQLite database with path remapping and dry-run support
- **Metadata source (TVDB)** — search series by title, fetch full episode lists via TVDB v4 API with JWT auth
- **RefreshSeriesMetadata** — command handler that syncs episodes from TVDB into the local library
- **RSS sync pipeline** — automatic 15-min RSS feed polling → parse → series match → decision engine evaluation → ranked grab via download client
- **Grab service** — picks the right download client by protocol and priority, sends releases, records history
- **History tracking** — records grab events per episode for duplicate detection
- **Import pipeline** — completed downloads scanned, parsed, matched to episodes, moved/hardlinked into series library with configurable naming tokens
- **File organizer** — naming token system for episode filenames (series title, season, episode, quality, release group)
- **Sonarr v3 API** — 35+ endpoints with wire-compatible JSON: series CRUD, episodes, episode files, quality profiles, quality definitions, custom formats, commands, history (paged), calendar, indexer/download client/notification CRUD + schema, root folders, parse, health, wanted/missing, system status
- **Root folders + file browser** — `/api/v3/filesystem` directory-listing endpoint and `POST`/`DELETE /api/v3/rootfolder` CRUD backing a `FileBrowserModal` on the Settings → Media Management page. Existing series' implicit root paths are persisted into a `root_folders` table on first boot.
- **Library Import** — Series → Library Import scans a root folder, auto-matches each sub-folder against TVDB (with a concurrency cap), and bulk-imports series with per-row Quality Profile / Monitor mode / Season Folder / Series Type overrides. Requires a TVDB API key configured in Settings → General.
- **Indexers + Download Clients settings pages** — schema-driven add/edit/delete UI for all 10 indexer and 20 download-client providers. Download Clients page also has a Remote Path Mappings sub-panel. Test / Test All actions remain deferred pending per-provider test endpoints on the backend.
- **Connect (notifications) settings page** — schema-driven add/edit/delete for all 24 notification providers with OnGrab / OnDownload / OnHealthIssue event triggers + tag binding. The full 13-event Sonarr trigger set will arrive when the backend expands its supported events.
- **Import Lists + Metadata settings (preview)** — read-only catalogs listing the 11 Import List providers and 4 Metadata consumers that sonarr2 has registered. Persisted add/edit/delete arrives in a follow-up sub-project once backend stores land.
- **API v6** — clean REST surface alongside v3: cursor pagination, RFC 9457 error envelopes, ~48 endpoints for series, episodes, profiles, commands, history, providers, notifications
- **API key authentication** — `X-Api-Key` header or `?apikey=` query param, matching Sonarr's convention
- **Filesystem watcher** — fsnotify-based monitoring with 2-second debouncing; detects changes in series folders for instant scan
- **Download monitoring** — 1-minute polling of download clients for completed items, auto-triggers import
- **Docker-ready** — multi-stage Dockerfile producing an Alpine-based image (~25 MB) with a `PUID`/`PGID` entrypoint so bind-mounted `/config` and `/data` volumes inherit host-side ownership. LinuxServer.io-style conventions.
- **CI** — GitHub Actions for lint (staticcheck + golangci-lint) and test (race detector + Postgres testcontainers)
- **Real-time push** — SignalR WebSocket transport (Sonarr-compatible) and Server-Sent Events; live updates for series, episodes, commands, and queue changes
- **Web UI** — React + TypeScript + Vite dark-themed frontend with series list (progress bars, status badges), series detail with season/episode tables, weekly calendar, activity queue/history with live refresh, wanted/missing episodes, system status with health checks, settings for indexers/download clients/profiles, connection status indicator
- **Ops hardening** — security headers (X-Frame-Options, nosniff), permissive CORS for API-key auth, per-IP rate limiting (30 req/s sustained, 100 burst), URLBase routing for reverse proxy support
- **Release engineering** — GitHub Actions release workflow producing multi-arch binaries (linux/amd64, linux/arm64, linux/arm/v7, macOS) and Docker images on tag push; built-in update checker polls GitHub Releases API with 24-hour cache and surfaces results as a health notice

### What's NOT yet implemented

This project covers the full planned roadmap (M0–M23). The areas still short of a production v1 release are end-user hardening: comprehensive integration tests, a polished UI, and real-world validation against large libraries. See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) for architectural details.

## Quick Start

```bash
# Build (frontend + backend)
make build

# Run with SQLite (default)
./dist/sonarr2 -port 8989 -bind 0.0.0.0

# Run with Docker
docker build -f docker/Dockerfile -t sonarr2 .
docker run -d --name sonarr2 \
  -p 8989:8989 \
  -e PUID=$(id -u) \
  -e PGID=$(id -g) \
  -e TZ=America/Chicago \
  -v ./config:/config \
  -v /path/to/tv:/data \
  sonarr2
```

### Docker notes

- **`PUID` / `PGID`.** The container runs as `1000:1000` by default; set `PUID`/`PGID` env vars to match the host user that owns your media library. The entrypoint drops privileges to that uid:gid via `su-exec` before launching the binary, so bind-mounted `/config` and `/data` inherit the expected ownership. Pass `-e PUID=$(id -u) -e PGID=$(id -g)` for a "same user as the host" setup.
- **Volumes.** `/config` holds sonarr2's own state (DB + logs + backups) — the entrypoint ensures it's writable by PUID:PGID. `/data` is your media library — bind-mount it from the host and make sure the host-side permissions already allow PUID:PGID to read/write. We deliberately don't recursively chown `/data` at boot because it can be terabytes.
- **Timezone.** The image includes `tzdata`; set `TZ=<Area/City>` so scheduled tasks fire at the times you expect.
- **Filesystem visibility.** The `/api/v3/filesystem` endpoint only sees what the process sees inside the container, so Library Import won't find folders that aren't bind-mounted.

## Development

```bash
git clone https://github.com/ajthom90/sonarr2.git
cd sonarr2
make test          # run all tests with race detector
make lint          # gofmt + go vet
make build         # build frontend then produce dist/sonarr2
make build-backend # build Go binary only (skips npm)
make frontend      # build frontend only (cd frontend && npm ci && npm run build)

# Frontend dev server (hot-reload, proxies /api/* to :8989)
cd frontend && npm run dev
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## Architecture

- **Go 1.23** backend — single static binary, ~15MB, frontend embedded via `//go:embed`
- **React 18 + TypeScript + Vite** frontend — dark theme, SPA with client-side routing; `npm run dev` proxies to the Go backend for local development
- **Postgres-first** with SQLite support — no "database is locked" errors via application-level single-writer discipline
- **GPL-3 licensed** — compatible with upstream Sonarr (also GPL-3); see [LICENSE](./LICENSE) and [NOTICE](./NOTICE)
- **Multi-arch** — builds for linux/amd64, linux/arm64, linux/arm/v7, macOS

See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) for the full architectural spec.

## License

GPL-3.0 — see [LICENSE](./LICENSE) and [NOTICE](./NOTICE). This matches upstream Sonarr's license.
