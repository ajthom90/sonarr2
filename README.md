# sonarr2

A feature-complete rewrite of [Sonarr](https://github.com/Sonarr/Sonarr) focused on performance for large TV libraries.

## Current Status

**Milestone 16 of 24 complete** — the core backend is functional with a fully connected React frontend and all remaining providers wired up. Not yet ready for end users.

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
- **Metadata source (TVDB)** — search series by title, fetch full episode lists via TVDB v4 API with JWT auth
- **RefreshSeriesMetadata** — command handler that syncs episodes from TVDB into the local library
- **RSS sync pipeline** — automatic 15-min RSS feed polling → parse → series match → decision engine evaluation → ranked grab via download client
- **Grab service** — picks the right download client by protocol and priority, sends releases, records history
- **History tracking** — records grab events per episode for duplicate detection
- **Import pipeline** — completed downloads scanned, parsed, matched to episodes, moved/hardlinked into series library with configurable naming tokens
- **File organizer** — naming token system for episode filenames (series title, season, episode, quality, release group)
- **Sonarr v3 API** — 35+ endpoints with wire-compatible JSON: series CRUD, episodes, episode files, quality profiles, quality definitions, custom formats, commands, history (paged), calendar, indexer/download client/notification CRUD + schema, root folders, parse, health, wanted/missing, system status
- **API v6** — clean REST surface alongside v3: cursor pagination, RFC 9457 error envelopes, ~48 endpoints for series, episodes, profiles, commands, history, providers, notifications
- **API key authentication** — `X-Api-Key` header or `?apikey=` query param, matching Sonarr's convention
- **Filesystem watcher** — fsnotify-based monitoring with 2-second debouncing; detects changes in series folders for instant scan
- **Download monitoring** — 1-minute polling of download clients for completed items, auto-triggers import
- **Docker-ready** — multi-stage Dockerfile producing a ~20MB distroless static binary
- **CI** — GitHub Actions for lint (staticcheck + golangci-lint) and test (race detector + Postgres testcontainers)
- **Real-time push** — SignalR WebSocket transport (Sonarr-compatible) and Server-Sent Events; live updates for series, episodes, commands, and queue changes
- **Web UI** — React + TypeScript + Vite dark-themed frontend with series list (progress bars, status badges), series detail with season/episode tables, weekly calendar, activity queue/history with live refresh, wanted/missing episodes, system status with health checks, settings for indexers/download clients/profiles, connection status indicator

### What's NOT yet implemented

Remaining provider implementations, migration tool, and more. See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) for the full roadmap.

## Quick Start

```bash
# Build (frontend + backend)
make build

# Run with SQLite (default)
./dist/sonarr2 -port 8989 -bind 0.0.0.0

# Run with Docker
docker build -f docker/Dockerfile -t sonarr2 .
docker run -p 8989:8989 -v ./config:/config sonarr2
```

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
- **Clean-room reimplementation** — MIT licensed, no Sonarr (GPL-3) source code copied
- **Multi-arch** — builds for linux/amd64, linux/arm64, linux/arm/v7, macOS

See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) for the full architectural spec.

## License

MIT — see [LICENSE](./LICENSE).
