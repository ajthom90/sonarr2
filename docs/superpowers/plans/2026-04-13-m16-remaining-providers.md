# Milestone 16 — Remaining Providers

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the remaining indexers, download clients, and notification providers for full Sonarr parity. M6 shipped Newznab + SABnzbd as reference implementations. M16 adds the other ~25 most-used providers. After M16, users can configure their real indexer/download client/notification stack.

**Architecture:** Each provider follows the M6 pattern: settings struct with form tags, implementation of the kind interface, httptest-based tests with canned responses. Providers are registered in app.New via `init()` or explicit registration.

---

## Scope

### Indexers (5 more, total 6 with Newznab)
- **Torznab** — torrent indexer via Newznab-compatible API (reuses Newznab parser, changes protocol to torrent)
- **IPTorrents** — RSS-only torrent indexer
- **Nyaa** — anime torrent RSS
- **BroadcastheNet** — private tracker API
- **TorrentRss** — generic torrent RSS feed

### Download Clients (5 more, total 6 with SABnzbd)
- **NZBGet** — Usenet download client (JSON-RPC API)
- **qBittorrent** — Torrent client (web API)
- **Transmission** — Torrent client (RPC API)
- **Deluge** — Torrent client (JSON-RPC API)
- **Blackhole** — Drop NZB/torrent files to a folder (no API)

### Notifications (8)
- **Discord** — webhook
- **Slack** — webhook
- **Telegram** — Bot API
- **Email** — SMTP
- **Webhook** — generic HTTP POST
- **Pushover** — push notification API
- **Gotify** — self-hosted push
- **CustomScript** — run an executable

---

## Task 1 — Notification provider interface + store

Add the notification provider kind (not yet in the SDK from M6).

**Files:**
- Create: `internal/providers/notification/notification.go` — interface, event types
- Create: `internal/providers/notification/registry.go`
- Create: `internal/providers/notification/store.go` + both dialect stores
- Migration: `00015_notifications.sql`
- Queries + sqlc regen

### Notification interface

```go
type Notification interface {
    providers.Provider
    OnGrab(ctx context.Context, msg GrabMessage) error
    OnDownload(ctx context.Context, msg DownloadMessage) error
    OnHealthIssue(ctx context.Context, msg HealthMessage) error
}

type GrabMessage struct {
    SeriesTitle string
    EpisodeTitle string
    Quality string
    Indexer string
}

type DownloadMessage struct {
    SeriesTitle string
    EpisodeTitle string
    Quality string
}

type HealthMessage struct {
    Type string
    Message string
}
```

Commit: `feat(providers/notification): add Notification interface, registry, and store`

---

## Task 2 — Torznab + torrent indexers

**Torznab** (`internal/providers/indexer/torznab/`):
- Extends Newznab — same XML format, different protocol (torrent)
- Adds `minSeeders` setting
- Parses `<torznab:attr name="seeders">` and `<torznab:attr name="peers">`
- Reuse the Newznab XML parser with torrent-specific extensions

**TorrentRss** (`internal/providers/indexer/torrentrss/`):
- Generic RSS parser for torrent sites
- Settings: feedUrl, cookie (optional)
- Parses standard RSS `<enclosure>` for download URL

**IPTorrents, Nyaa, BroadcastheNet** — stubs with settings structs and Test() methods. Full search implementations are complex (each tracker has unique API quirks) — M16 ships settings + RSS + Test; full search can be added per-tracker in follow-up PRs.

Commit: `feat(providers/indexer): add Torznab, TorrentRss, and tracker stubs`

---

## Task 3 — Download clients

**NZBGet** (`internal/providers/downloadclient/nzbget/`):
- JSON-RPC over HTTP: `POST /jsonrpc` with `{"method":"append","params":[...]}`
- Settings: host, port, username, password, category, useSsl
- Methods: Add (append), Items (listgroups), Remove (editqueue delete), Status (status), Test (version)

**qBittorrent** (`internal/providers/downloadclient/qbittorrent/`):
- Web API: login via `POST /api/v2/auth/login`, then session cookie
- Settings: host, port, username, password, category, useSsl
- Add: `POST /api/v2/torrents/add` (multipart form with torrent URL)
- Items: `GET /api/v2/torrents/info`
- Remove: `POST /api/v2/torrents/delete`

**Transmission** (`internal/providers/downloadclient/transmission/`):
- RPC: `POST /transmission/rpc` with JSON body, requires `X-Transmission-Session-Id` header (409 → retry with header from response)
- Settings: host, port, username, password, category, useSsl
- Methods: torrent-add, torrent-get, torrent-remove

**Deluge** (`internal/providers/downloadclient/deluge/`):
- JSON-RPC: `POST /json` with `{"method":"core.add_torrent_url","params":[...]}`
- Settings: host, port, password, category, useSsl

**Blackhole** (`internal/providers/downloadclient/blackhole/`):
- No API — writes NZB/torrent files to a configured folder
- Settings: nzbFolder, torrentFolder
- Add: write the download URL content to a file in the folder
- Items/Remove/Status: no-ops (blackhole doesn't track state)

All clients follow the same pattern: settings struct, constructor taking settings + *http.Client, httptest-based tests.

Commit: `feat(providers/downloadclient): add NZBGet, qBittorrent, Transmission, Deluge, Blackhole`

---

## Task 4 — Notification providers

**Discord** — POST to webhook URL with embed JSON
**Slack** — POST to webhook URL with attachment JSON
**Telegram** — POST to `https://api.telegram.org/bot{token}/sendMessage`
**Email** — SMTP via `net/smtp` (add `SONARR2_SMTP_*` config later; M16 just ships the provider)
**Webhook** — POST to configurable URL with JSON body
**Pushover** — POST to `https://api.pushover.net/1/messages.json`
**Gotify** — POST to `{url}/message` with token
**CustomScript** — `exec.Command` with environment variables

Each implements `notification.Notification`. Tests use httptest for HTTP-based providers. CustomScript test runs `echo` or a no-op script.

Commit: `feat(providers/notification): add Discord, Slack, Telegram, Email, Webhook, Pushover, Gotify, CustomScript`

---

## Task 5 — Register all providers in app + notification dispatch

Wire all new providers into app.New:
- Register indexer factories: Torznab, TorrentRss, IPTorrents, Nyaa, BroadcastheNet
- Register download client factories: NZBGet, qBittorrent, Transmission, Deluge, Blackhole
- Register notification factories: all 8
- Create notification instance store + dispatch service

**Notification dispatch:** Subscribe to domain events and call matching notification instances:
```go
events.SubscribeAsync[grab.ReleasesGrabbed](bus, func(ctx context.Context, e grab.ReleasesGrabbed) {
    // Load enabled notification instances, call OnGrab for each
})
```

Add v3 + v6 API endpoints for notifications (CRUD + schema + test).

Commit: `feat(app): register all providers and wire notification dispatch`

---

## Task 6 — README + push

- Bump to M16
- Add provider counts to implemented list
- Push + CI

---

## Done

After Task 6, sonarr2 supports 6 indexers, 6 download clients, and 8 notification providers. Users can configure their real setup through the settings UI. The notification dispatch fires on grab/download/health events automatically.
