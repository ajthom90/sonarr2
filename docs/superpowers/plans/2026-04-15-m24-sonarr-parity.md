# M24 Overnight Implementation Log

**Date:** 2026-04-15 (overnight session)
**Branch:** `claude/review-milestones-progress-Kf4lo`
**Design:** [M24 Design Doc](../specs/2026-04-15-m24-sonarr-parity-design.md)

## Purpose of this doc

A commit-by-commit map of what landed overnight, so codex's morning review
has an audit trail to cross-check against the git log. Every line here
corresponds to one or more commits on the branch.

## Final state

- **Commits:** 19 new (plus earlier relicense + gap-analysis commits)
- **Tests:** 69 packages, all passing
- **Build:** Go + frontend (vite) both clean
- **Pushes:** Every commit pushed to origin as it landed

## What landed, in commit order

### 1. Foundational subsystems (tags, blocklist, remote paths, recycle bin)

| Commit | Scope |
|---|---|
| `873a0fe` | Tags: migration 00018 + sqlc + `internal/tags` + `/api/v3/tag` + `/tag/detail` with case-insensitive unique labels |
| `f1fbcc6` | Blocklist: migration 00019 + paged store + `/api/v3/blocklist` (list+get+delete+bulk-delete) |
| `50cb303` | Remote Path Mappings: migration 00020 + store + `/api/v3/remotepathmapping` + `Translate()` with separator-boundary prefix matching |
| `18218e6` | Recycle Bin: migration 00021 (adds `recycle_bin` + `recycle_bin_cleanup_days` to host_config) + `internal/recyclebin` with DeleteFile/DeleteFolder/Cleanup/Empty |

### 2. Navigation shell

| Commit | Scope |
|---|---|
| `ac85923` | Sidebar restructured to Sonarr's 6-item layout with nested sub-items; all missing routes (Activity × 3, Wanted × 2, Settings × 13, System × 6, Library Import / Mass Editor / Season Pass) scaffolded as first-class pages. Blocklist page is fully wired to `/api/v3/blocklist`; Tags page is wired to `/api/v3/tag/detail`; other settings pages render a `PagePlaceholder` describing what's pending |

### 3. Profiles

| Commit | Scope |
|---|---|
| `4fb19dc` | Release Profiles + Delay Profiles: migration 00022 + sqlc + `internal/releaseprofile` (Match with required/ignored + regex) + `internal/delayprofile` (ApplicableProfile tag matching) + `/api/v3/releaseprofile` + `/api/v3/delayprofile` + seed default delay profile |

### 4. Calendar feed

| Commit | Scope |
|---|---|
| `da46ffa` | iCalendar feed at `/feed/calendar/Sonarr.ics` and `/feed/v3/calendar/Sonarr.ics` (both paths) with `pastDays` / `futureDays` / `unmonitored` / `premieresOnly` / `asAllDay` / `tags` query params. Hand-written VCALENDAR serializer (no external dep) with proper RFC 5545 line folding and escaping |

### 5. Provider expansion

| Commit | Scope |
|---|---|
| `2e0f355` | 13 new notification providers: PushBullet, Ntfy, Kodi/XBMC, Plex, Emby, Notifiarr, Prowl, Apprise, Join, Simplepush, Pushcut, Mailgun, SendGrid, Signal. Plus shared `httputil.go` (PostJSON, PostJSONWithHeaders, PostForm, Get, Put) |
| `007d8f9` | 3 notification stubs (Twitter/Trakt/SynologyIndexer) + 14 download-client providers: Aria2 (full JSON-RPC), split Blackhole into UsenetBlackhole + TorrentBlackhole, plus stubs for NzbVortex, Pneumatic, DownloadStation (torrent + usenet), rTorrent, uTorrent, Vuze, Hadouken, Flood, FreeboxDownload, Tribler, RQBit. New `internal/providers/downloadclient/stubhelper.go` with Stub / StubUsenet / StubTorrent base types |
| `2667dd3` | 4 new indexer stubs: FileList, HDBits, Torrentleech, Fanzub — brings indexer total to 10/10 |

### 6. Health checks

| Commit | Scope |
|---|---|
| `ec3fc93` | 10 new health checks: ApiKeyValidation, AppDataLocation, Mount, SystemTime, RecyclingBin, RemotePathMapping, Proxy, RemovedSeries, ImportListStatus, NotificationStatus. Callback-based construction keeps the `internal/health` package decoupled from the rest of the codebase |

### 7. Metadata consumers (entire subsystem)

| Commit | Scope |
|---|---|
| `b784b74` | Metadata consumer subsystem (`internal/metadata` + Registry pattern) + 4 consumers: Kodi (XbmcMetadata — full tvshow.nfo + episode.nfo with unique IDs, actors, genres, episode guide URL, thumb), Plex (MediaBrowserMetadata), Roksbox (RoksboxMetadata — .xml video file), WDTV (WdtvMetadata — details.xml). All matching Sonarr's Implementation identifiers byte-for-byte |

### 8. Import Lists (entire subsystem)

| Commit | Scope |
|---|---|
| `c9c7444` | Import Lists subsystem: migration 00023 (import_lists + import_list_exclusions), `internal/importlist` with ListProvider interface + Instance/Exclusion/Monitor types + Registry + Store interface. 11 provider stubs with full Sonarr-compatible Settings schemas: AniList, MyAnimeList, Plex Watchlist, Plex RSS, generic RSS, Simkl, Sonarr-to-Sonarr, Trakt User/List/Popular, Custom |

### 9. Auto Tagging + Scene Mapping

| Commit | Scope |
|---|---|
| `8249dd7` | Auto Tagging: migration 00024 + `internal/autotag` with Rule/Specification/SeriesAttr + 7 spec types (Genre/SeriesStatus/SeriesType/Network/OriginalLanguage/Year/RootFolder) + Matches/ApplyTags/RemoveTags. Scene Mapping: migration 00025 + `internal/scenemapping` with LookupSceneSeason + LookupTvdbIDByTitle with Sonarr-compatible title normalization |

### 10. v3 API expansion

| Commit | Scope |
|---|---|
| `4116618` | `/api/v3/metadata` + `/api/v3/importlist` (+ `/importlistexclusion`) + `/api/v3/autotagging` — list stubs (empty arrays) + /schema endpoints that enumerate the new registries with full Sonarr-compatible field shapes |
| `6a8fa56` | `/api/v3/release` (Interactive Search list + grab + push) + `/api/v3/manualimport` (folder scan + execute) — endpoint shells returning valid JSON so frontend modals load; real implementations pending |

### 11. Documentation

| Commit | Scope |
|---|---|
| `78ea30c` | M24 design doc (`docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md`) with subsystem status table, provider counts, architectural decisions, codex-review checklist, out-of-scope list. README updated from "all milestones complete" to "M0–M23 complete; M24 in progress" with link to design doc |

## Parity scoreboard

| Category | Before | After | Sonarr upstream |
|---|---|---|---|
| Sidebar items visible | 7 flat | 6 nested + all sub-items | 6 nested |
| Indexer identifiers | 6 | 10 | 10 |
| Download client identifiers | 6 | 20 | 20 |
| Notification identifiers | 8 | 24 (3 stubs) | 25 (Boxcar dropped) |
| Metadata consumers | 0 | 4 | 4 |
| Import List providers | 0 | 11 | 11 |
| Auto-tag spec types | 0 | 7 | 7 |
| Health checks | 7 | 17 | ~25 |
| Release Profiles | — | ✓ | ✓ |
| Delay Profiles | — | ✓ | ✓ |
| Tags | stub | ✓ | ✓ |
| Blocklist | — | ✓ | ✓ |
| Remote Path Mappings | — | ✓ | ✓ |
| Recycle Bin | — | core done | ✓ |
| Scene Mapping | — | core done | ✓ |
| iCal feed | — | ✓ | ✓ |
| Test count | 61 | 69 | n/a |

## Known gaps for codex to flag

Not landed overnight — documented here so the review catches them:

1. **Subsystem integration** with existing pipeline:
   - Recycle Bin: package + config present, but episode-file deletion callsites (`library.EpisodeFilesStore.Delete`, housekeeping) don't yet call `recyclebin.DeleteFile` / `recyclebin.DeleteFolder`.
   - Remote Path Mappings: `Translate()` exists but is not yet called by the importer when resolving download-client-reported paths.
   - Release Profiles: `Match()` is not yet called by the decision engine.
   - Delay Profiles: `ApplicableProfile` + `ProtocolDelay` are not yet consulted by the grab pipeline.
   - Blocklist: `Matches()` is not yet called by the decision engine.
   - Metadata consumers: registered but not yet hooked into library refresh / episode-file-import events.
   - Import List sync: no scheduled task (`ImportListSyncCommand`) implemented; providers all return ErrStub.
   - Auto-tagging: rule engine exists but is not invoked on series add / refresh.
   - Scene mapping: store interface exists but no scheduled fetcher populates it.

2. **Store implementations** for new subsystems:
   - `internal/importlist`, `internal/autotag`, `internal/scenemapping` have Store interfaces but no sqlc/SQLite/Postgres implementations. Migrations are in place so the tables exist; the Go store code needs to follow the pattern from `internal/tags` / `internal/blocklist` / `internal/releaseprofile`.

3. **Scheduled tasks** pending:
   - `ImportListSyncCommand` (5 min)
   - `UpdateSceneMappingCommand` (3 h)
   - `ApplicationUpdateCheckCommand` (6 h, currently only a health check)
   - `CleanUpRecycleBinCommand` (24 h — calls `recyclebin.Cleanup` using configured `recycle_bin_cleanup_days`)

4. **On-demand commands** pending:
   - Rescan, Move, BulkMove, RenameFiles, RenameSeries, DeleteSeriesFiles
   - EpisodeSearch / SeasonSearch / SeriesSearch (per-interactive-search)
   - MissingEpisodeSearch / CutoffUnmetEpisodeSearch (bulk)
   - DownloadedEpisodesScan / TestProvider

5. **v3 wire-compat verification against Sonarr's OpenAPI spec** — I added endpoints matching Sonarr's field names on new writes, but did not run an automated byte-diff against `.sonarr-upstream/src/Sonarr.Api.V3/*Resource.cs`. Worth a follow-up sweep.

6. **Provider fleshing** — 13 of 20 DL clients, 7 of 10 indexers, 3 of 24 notifications, and 11 of 11 import lists are stubs. Each stub carries its Sonarr-compatible Settings schema so the UI works, but actions return ErrStub.

7. **Quality definition editor** — PUT `/api/v3/qualitydefinition/{id}` and the store.Update method are not yet added.

8. **Custom Format specification types** — still only regex (ReleaseTitleSpecification). Missing: Language, IndexerFlag, Source, Resolution, ReleaseGroup, Size, ReleaseType.

9. **General Settings sub-sections** — Host / Security / Proxy / Logging / Analytics / Updates panels pending (the General page today still shows only TVDB API key + API key display).

10. **UI Settings page** — currently a PagePlaceholder; theme / language / date format persistence endpoint is not yet added.

## What codex should verify

1. `git log --oneline origin/claude/review-milestones-progress-Kf4lo` — 19 new commits in chronological order matching the table above.
2. `go test ./...` — all 69 packages pass on `develop`.
3. `go build ./...` — clean.
4. `cd frontend && npm run build` — clean (tsc + vite).
5. `grep -r "SPDX-License-Identifier" internal/ | wc -l` — ported files carry GPL-3 headers.
6. `ls internal/db/migrations/sqlite/` — migrations numbered 00018 through 00025, append-only; no existing migration modified.
7. Cross-reference `internal/app/app.go` notification + DL client + indexer + metadata + importlist registry blocks against the upstream `src/NzbDrone.Core/*/Registry` equivalents to confirm identifier names.
8. `diff <(grep -r "package " internal/providers/notification --include=*.go | grep -v _test | cut -d: -f2- | sort -u) <(ls -d .sonarr-upstream/src/NzbDrone.Core/Notifications/*/ | xargs -n1 basename | sort)` — compare sonarr2 notification package count vs Sonarr's.
