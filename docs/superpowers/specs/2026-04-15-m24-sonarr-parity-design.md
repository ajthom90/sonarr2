# M24 — Sonarr Feature Parity (Drop-In Replacement)

**Date:** 2026-04-15
**Status:** In progress
**Authors:** Claude (drafted overnight), AJ Thompson (review + codex audit)

## 1. Goal

Bring sonarr2 to full feature parity with upstream Sonarr (v4/v5) so it can
serve as a **drop-in replacement** without breaking existing integrations
(Overseerr, Prowlarr, Notifiarr, nzb360, LunaSea, Home Assistant). Every
sidebar menu item in Sonarr should resolve to an equivalent page in sonarr2;
every provider identifier in Sonarr's `/api/v3/{indexer,downloadclient,
notification}/schema` should exist in sonarr2's schema with the same
JSON field names; and every core behavior (Recycle Bin on delete, Release
Profile term filtering, Delay Profile grab throttling, Tags cross-binding,
Blocklist rejection) should match.

## 2. Scope

This milestone spans many independent subsystems. To avoid one giant
landing, it's staged across multiple commits on the
`claude/review-milestones-progress-Kf4lo` branch. The status table below
tracks completeness.

### 2.1 Foundational subsystems

| Feature | Status | Notes |
|---|---|---|
| Tags (CRUD + cross-binding) | ✅ migration + store + `/api/v3/tag` + `/tag/detail` | Cross-resource linking (series/indexers/DL clients/notifications/etc.) will be wired when those subsystems grow `tags` columns. |
| Blocklist | ✅ migration + paged store + `/api/v3/blocklist` | Decision-engine rejection spec pending. |
| Remote Path Mappings | ✅ migration + store + `/api/v3/remotepathmapping` + Translate() | Import pipeline will call Translate() when picking up completed downloads. |
| Recycle Bin | ✅ config + `internal/recyclebin` + `CleanUpRecycleBin` helper | Integration with episode-file deletion callsites pending. |
| Release Profiles | ✅ migration + store + `/api/v3/releaseprofile` + Match() | Decision engine integration pending. |
| Delay Profiles | ✅ migration + store + `/api/v3/delayprofile` + ApplicableProfile() + seed default | Grab pipeline integration pending. |
| Auto Tagging | ❌ | Rule engine and UI pending. |

### 2.2 Navigation

| Feature | Status |
|---|---|
| Sidebar restructure to Sonarr's 6-item nav | ✅ |
| Series sub-items: Add New / Library Import / Mass Editor / Season Pass | ✅ Library Import shipped end-to-end (see §2.5); Add New persists `qualityProfileId`/`seasonFolder`/`monitorNewItems`/`addOptions` correctly but the form still hardcodes Season Folder / Monitor mode to defaults — explicit UI controls are a follow-up. Mass Editor / Season Pass remain scaffolded. |
| `/activity/queue` + `/activity/history` + `/activity/blocklist` | ✅ |
| `/wanted/missing` + `/wanted/cutoffunmet` | ✅ (CutoffUnmet is a placeholder pending /api/v3/wanted/cutoff) |
| 13 Settings sub-pages | ✅ (Tags wired end-to-end; others scaffolded as PagePlaceholder) |
| 6 System sub-pages (status/tasks/backup/updates/events/logs/files) | ✅ |

### 2.3 Provider expansion

| Category | Count | Status |
|---|---|---|
| Indexers | 10/10 | All Sonarr identifiers registered; 3 full (Newznab/Torznab/TorrentRss), 7 stubs (IPTorrents, Nyaa, BroadcastheNet, FileList, HDBits, Torrentleech, Fanzub) |
| Download Clients | 20/20 | 7 full (SABnzbd, NZBGet, qBittorrent, Transmission, Deluge, Blackhole×3, Aria2); 13 stubs (NzbVortex, Pneumatic, Synology DS ×2, rTorrent, uTorrent, Vuze, Hadouken, Flood, FreeboxDownload, Tribler, RQBit) |
| Notifications | 24/25 | 21 full (Discord, Slack, Telegram, Email, Webhook, Pushover, Gotify, CustomScript, PushBullet, Ntfy, Kodi/Xbmc, Plex, Emby, Notifiarr, Prowl, Apprise, Join, Simplepush, Pushcut, Mailgun, SendGrid, Signal); 3 stubs (Twitter, Trakt, SynologyIndexer). Boxcar is deprecated and not included. |
| Metadata Consumers | 0/4 | Kodi/Plex/Roksbox/WDTV pending — not yet started. |
| Import List providers | 0/8 | AniList/MAL/Plex Watchlist/Plex RSS/RSS/Simkl/Sonarr/Trakt pending — not yet started. |

### 2.4 Calendar / feeds

| Feature | Status |
|---|---|
| `/feed/calendar/Sonarr.ics` iCalendar feed | ✅ with pastDays/futureDays/tags/unmonitored/premieresOnly/asAllDay params |

### 2.5 Pending subsystems

**Shipped since initial draft:**

- **Root folders + Library Import** — ✅ DONE. Backend: `/api/v3/filesystem`
  directory listing, `/api/v3/rootfolder` CRUD (`POST`/`DELETE`),
  `/api/v3/libraryimport/scan`, and extended `POST /api/v3/series` with
  `addOptions`. Frontend: new `FileBrowserModal` and `SearchOverrideModal`,
  rewritten `/add/import` page with per-row Quality Profile / Monitor mode /
  Season Folder / Series Type overrides, upgraded Media Management settings,
  and Add Series migrated to consume the persisted root-folder list. Existing
  series' implicit root paths are back-filled into a `root_folders` table on
  first boot. See
  [`docs/superpowers/plans/2026-04-14-root-folders-library-import.md`](../plans/2026-04-14-root-folders-library-import.md)
  for the full implementation plan.

- **Indexers + Download Clients settings pages** — ✅ DONE. Frontend-only
  (backend was already wired). Shared `ProviderListSection`,
  `ProviderPickerModal`, `ProviderSettingsModal`, and `SchemaFormField`
  components driven by the existing `/api/v3/indexer/schema` +
  `/api/v3/downloadclient/schema` endpoints. Two thin page wrappers
  (`/settings/indexers`, `/settings/downloadclients`) plus a
  `RemotePathMappingsPanel` bundled into the Download Clients page. Test /
  Test All actions are deferred pending per-provider test endpoints. See
  [`docs/superpowers/plans/2026-04-14-provider-settings-pages.md`](../plans/2026-04-14-provider-settings-pages.md)
  for the full implementation plan.

- **Connect + Import Lists + Metadata settings pages** — ✅ DONE (with
  scope-limited caveats). Connect is full CRUD against `/api/v3/notification`
  (24 providers, OnGrab / OnDownload / OnHealthIssue triggers, tag binding).
  Import Lists (`/settings/importlists`) and Metadata (`/settings/metadata`)
  landed as browsable read-only catalogs showing the 11 registered list
  providers and 4 registered metadata consumers, respectively. Persisted
  instances for those two backends are deferred — a subsequent sub-project
  will add `/api/v3/importlist` POST/PUT/DELETE plus a new metadata instance
  store. Notification backend only emits 3 of Sonarr's 13 event triggers
  today; expanding that is also a follow-up. See
  [`docs/superpowers/specs/2026-04-14-connect-importlist-metadata-pages-design.md`](2026-04-14-connect-importlist-metadata-pages-design.md)
  for the full design + scope note.

The following are targeted for completion in follow-up commits. Each is
non-trivial but isolated:

- **Import Lists** — table + CRUD + sync scheduler + exclusions + sync levels + 8 providers
- **Metadata consumers** — .nfo + image writers for Kodi/Plex/Roksbox/WDTV + Extras/subtitle import
- **Interactive Search** — `/api/v3/release` endpoint + modal UI
- **Manual Import** — `/api/v3/manualimport` endpoint + modal UI
- **Scene Mappings** — anime absolute-number mapping fetch + apply
- **Quality definition editor** — PUT `/api/v3/qualitydefinition/{id}` + UI
- **Custom Format specification types** (8 total — currently regex-only) — Language, IndexerFlag, Source, Resolution, ReleaseGroup, Size, ReleaseType
- **Release profile & delay profile decision-engine integration**
- **Missing health checks** (19 new: ApiKey, AppData, Proxy, Mount, SystemTime, etc.)
- **UI Settings** page (theme/language/date formats) + persistence
- **General Settings sub-sections** (Host/Security/Proxy/Logging/Analytics/Updates)
- **Series metadata config** — propers/repacks, retention, recycle-bin wiring, MediaInfo, Extras imports, script import, series types
- **Scheduled tasks** — ImportListSync, UpdateSceneMapping, ApplicationUpdateCheck, CleanUpRecycleBin
- **On-demand commands** — Rescan, Move, BulkMove, RenameFiles, Episode/Season/SeriesSearch, DownloadedEpisodesScan, TestProvider
- **Expanded v3 API surface** — releaseprofile (✅), delayprofile (✅), remotepathmapping (✅), blocklist (✅), filesystem (✅), rootfolder CRUD (✅), libraryimport/scan (✅) are done; manualimport / release / rename / queue/bulk / queue/grab / customfilter / autotagging / log / log/file / update / mediacover / localization / config/{naming,mediamanagement,host,ui,downloadclient,indexer} are pending

## 3. Licensing

Earlier in the session the repository was relicensed from MIT to GPL-3.0
so code may be ported directly from upstream Sonarr with attribution. All
migration SQL files and controller-port Go files carry SPDX headers and
cite the upstream path. `NOTICE` records the attribution at the project
level. See `CONTRIBUTING.md` §License.

## 4. Architectural decisions

### 4.1 Placeholder pages vs empty 404s

For scaffolded routes without a backend, the frontend renders a
`PagePlaceholder` component describing what's pending rather than 404ing.
This keeps the UX honest and means the sidebar's 6×N navigation tree
always resolves to a visible page, matching Sonarr's menu structure 1:1.

### 4.2 Provider stubs register identifiers

Every Sonarr provider identifier (e.g. `UTorrent`, `NzbVortex`,
`FileList`, `Twitter`) has a registered provider in sonarr2 even when the
implementation is a stub. The Settings struct carries the full Sonarr
schema (byte-identical JSON field names), so /schema responses and
migrated provider rows round-trip cleanly. Stubs return a clear
"not yet implemented" error from action methods; the Test endpoint
returns nil for indexer stubs (to allow saving) and errors for DL-client
and notification stubs.

### 4.3 Sidebar-level sub-items (5-level hierarchy)

Sonarr uses a two-level sidebar: top-level item + optional sub-items.
sonarr2 now matches that exactly. The Settings sidebar has 13 sub-items;
System has 6; Activity has 3; Wanted has 2; Series has 4 sub-items plus
the index itself.

### 4.4 Dual-dialect store pattern

Every new subsystem follows the existing pattern: sqlc queries under
`internal/db/queries/{sqlite,postgres}/`, generated Go code under
`internal/db/gen/`, and a Store interface with `store_sqlite.go` +
`store_postgres.go` implementations. Single-writer discipline is
preserved (all SQLite writes funnel through pool.Write).

### 4.5 Homelab / older-hardware constraints

- No new CGo dependencies.
- No heavy external libs for iCal (hand-written VCALENDAR), base64
  (inline), or email formatting.
- Per-provider HTTP helper (`internal/providers/notification/httputil.go`)
  shares one `http.Client` default so goroutines are bounded.

## 5. Test strategy

Every new subsystem ships with unit tests:
- Tags (CRUD, normalization, duplicate detection)
- Blocklist (CRUD, pagination, bulk delete, Matches helper)
- Remote Path Mappings (CRUD + Translate edge cases including
  Windows-style paths, case-insensitive host, separator boundary)
- Recycle Bin (permanent delete, move-to-recycle, age-filtered cleanup,
  disabled cleanup, Empty)
- Release Profiles (CRUD, Match with required/ignored/regex)
- Delay Profiles (CRUD, seed-row presence, ApplicableProfile tag matching)

Plus the existing 65+ package tests remain green after every commit.

## 6. Follow-up work

This design doc and the accompanying commit trail on
`claude/review-milestones-progress-Kf4lo` are the working record of what
was landed overnight. Codex's morning review should check:

1. **Navigation completeness** — `git log` shows the sidebar commit; the
   frontend builds and every route resolves.
2. **Provider identifiers** — the Sonarr-compatible identifiers for
   indexer / download client / notification schemas.
3. **Migration safety** — new migrations are numbered 00018–00022 and
   append only; no existing migration is edited.
4. **No regressions** — `go test ./...` shows all packages green (count
   rose from 61 → 65+ with new package tests).
5. **License hygiene** — SPDX headers on ported files; `NOTICE` records
   upstream attribution.
6. **Stubs vs full** — stub providers are clearly marked in their package
   docstrings ("stub for..."); full implementations have no such marker.
7. **What's unfinished** — section 2.5 is the authoritative list.

## 7. Out-of-scope

Things explicitly not in M24:

- **Translations** beyond what Sonarr ships. sonarr2 frontend is
  English-only for now; UI Settings language picker will land with the
  UI Settings subsystem.
- **Mobile app.** Sonarr has a web UI only; sonarr2 matches.
- **Clustering / multi-instance.** Single-node only.
