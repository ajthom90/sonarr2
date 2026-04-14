# Root Folders + Library Import — Design

**Date:** 2026-04-14
**Status:** Approved (revised 2026-04-14 after codebase review — see §13)
**Sub-project:** #1 of the post-audit Sonarr-parity UI program
**Scope decision:** First shippable slice from the 34-item gap audit at `/tmp/sonarr2-ui-gap-audit.md`

## 1. Goal

Give users two things that are missing today and that block all onboarding:

1. A proper way to **add a root folder** (pick a directory from the server filesystem, not type a blind path).
2. A proper way to **import an existing library of series** from that root folder (scan sub-folders, auto-match to TVDB, bulk-add).

Both map to Sonarr behavior that users expect. Without these, a new sonarr2 install has no path into the library other than manually adding shows one by one via search — and the one-by-one path is also broken on the test instance because TVDB lookup returns 503.

## 2. Non-goals

- Remote Path Mappings (separate sub-project, Download Clients concern).
- Recurring/unattended imports (that's Import List sync — a different subsystem already stubbed).
- Per-season import granularity (Sonarr doesn't do this; whole-series granularity).
- Changing the importer's hardlink/move behavior (inherits whatever Media Management specifies).
- Fixing the TVDB-key configuration UI (already present at `/settings/general`; this spec just depends on it).
- Series Detail, Series Index toolbar, or any other audit item beyond the two flagged above.
- Upgrading the v6 API surface (`internal/api/v6/rootfolder.go`, etc.). v6 keeps its current derive-from-series behavior; this sub-project only extends v3. A follow-up sub-project will mirror these endpoints in v6.

## 3. Architecture overview

Three new backend surfaces, one reusable frontend component, one replaced page, and one upgraded settings panel:

```
┌────────────────────────────────────────────────────────────────┐
│                         Frontend                                │
│  ┌──────────────────────┐       ┌──────────────────────────┐   │
│  │ FileBrowserModal     │◄──────│ Media Management         │   │
│  │ (shared)             │       │ Add Root Folder button   │   │
│  └──────────┬───────────┘       └──────────────────────────┘   │
│             │                                                   │
│             │                   ┌──────────────────────────┐   │
│             └──────────────────►│ LibraryImport page       │   │
│                                 │ /add/import              │   │
│                                 └──────────┬───────────────┘   │
└────────────────────────────────────────────┼───────────────────┘
                                             │
  GET /filesystem                GET /libraryimport/scan
  POST /rootfolder               GET /rootfolder (enriched)
  DELETE /rootfolder/{id}        POST /series (existing, reused)

┌────────────────────────────────────────────────────────────────┐
│                         Backend                                 │
│  filesystem.go      rootfolder.go (ext.)   libraryimport.go     │
│                             │                      │            │
│                             ▼                      ▼            │
│                    root_folders table     series_lookup (reused)│
└────────────────────────────────────────────────────────────────┘
```

## 4. Data model

### 4.1 New `root_folders` table — migration 00026

New migration `internal/db/migrations/{postgres,sqlite}/00026_root_folders.sql`:

```sql
-- sqlite
CREATE TABLE root_folders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_root_folders_path ON root_folders(path);
```

```sql
-- postgres
CREATE TABLE root_folders (
    id          SERIAL PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_root_folders_path ON root_folders(path);
```

Deliberately minimal. `accessible` and `free_space` are computed on read via `os.Stat` and `syscall.Statfs_t`; a mount that disappears between calls is visible immediately instead of showing stale persisted state.

### 4.2 Extend `series` table — migration 00027

Library Import's per-row `Quality Profile` and `Season Folder` settings have no home in the current domain model — `library.Series` (see `internal/library/series.go`) doesn't carry them, and the v3 `seriesInput` struct silently drops them today. This sub-project fixes that.

```sql
-- both dialects; sqlite shown, postgres identical syntax
ALTER TABLE series ADD COLUMN quality_profile_id INTEGER
    REFERENCES quality_profiles(id) ON DELETE SET NULL;
ALTER TABLE series ADD COLUMN season_folder BOOLEAN NOT NULL DEFAULT 1;  -- postgres: DEFAULT TRUE
ALTER TABLE series ADD COLUMN monitor_new_items TEXT NOT NULL DEFAULT 'all';
CREATE INDEX idx_series_quality_profile_id ON series(quality_profile_id);
```

`monitor_new_items` supports values `all | none`. (Other Sonarr values like `newOnly`, `futureOnly` affect only "which *new* episodes should be monitored going forward" — scope-deferred to a follow-up; default `all` preserves current behavior.)

Correspondingly, `library.Series` gains three fields, the sqlc queries for series are updated, and `seriesHandler.toResource` now reads `QualityProfileID` from the series row instead of hardcoding `1`.

### 4.3 Startup-time backfill (not a migration hook)

Goose migrations in this repo are SQL-only (`internal/db/migrate.go`); adding a Go migration hook would diverge from the pattern. Instead, `internal/app/app.go` gains a post-migration idempotent backfill:

```go
// After db.Migrate succeeds, before the HTTP server starts:
if err := rootfolder.BackfillFromSeries(ctx, rootFolderStore, seriesStore); err != nil {
    return fmt.Errorf("rootfolder backfill: %w", err)
}
```

`BackfillFromSeries` reads distinct `filepath.Dir(series.path)` from the `series` table and inserts any not already present in `root_folders` using `INSERT ... ON CONFLICT (path) DO NOTHING`. Safe to call on every boot. Installations that pre-date this spec get their implicit roots persisted without user intervention.

## 5. Backend endpoints

### 5.1 `GET /api/v3/filesystem`

File: `internal/api/v3/filesystem.go`.

Query params:
- `path` (required) — absolute path to list
- `allowFoldersWithoutTrailingSlashes` (bool, default true) — Sonarr compat param
- `includeFiles` (bool, default false) — when true, files appear in `files[]`; directories always appear in `directories[]`

Response:
```json
{
  "parent": "/data",
  "directories": [
    {"type": "folder", "name": "tv", "path": "/data/tv"}
  ],
  "files": []
}
```

Security rules:
- Reject paths whose `filepath.Clean`ed form differs from the input or contains a `..` component after cleaning (blocks traversal, tolerates redundant separators)
- Reject paths that resolve outside allowed roots after `filepath.EvalSymlinks`
- Deny-list: `/proc`, `/sys`, `/dev`, `/.git` at any depth (read-only system dirs; won't help a user anyway)
- Return 403 on permission-denied so the UI can message correctly vs 500

### 5.2 `POST /api/v3/rootfolder`

File: extends `internal/api/v3/rootfolder.go`.

Request accepts the Sonarr-compat full rootFolderResource shape but only `path` is required; everything else is ignored on write:
```json
{ "path": "/data/tv" }
```

Behavior:
- Decode body; reject missing/empty `path` → 400 `{message: "path is required"}`
- `os.Stat` the path; if `!IsDir` or `os.IsNotExist` → 400 `{message: "folder does not exist: /data/tv"}`
- `filepath.Clean` the path for normalized storage
- Insert into `root_folders`; on unique-constraint violation → 409 `{message: "root folder already exists"}`
- Return 201 with the populated `rootFolderResource` (id, path, freeSpace via `statfs`, accessible=true, unmappedFolders=[])

### 5.3 `GET /api/v3/rootfolder` (upgrade)

Today: infers root folders from distinct `filepath.Dir(series.path)`.
After: lists rows from `root_folders` (the backfill from §4.3 guarantees derived roots are persisted on first boot). No more ad-hoc derivation on each request.

`freeSpace` and `accessible` computed per row on each call via `os.Stat` + `syscall.Statfs_t`. A slow mount would slow this response; acceptable for a settings page (user is there infrequently) and cacheable by frontend.

Response is the full Sonarr-compat resource shape:
```json
[
  { "id": 1, "path": "/data/tv", "freeSpace": 13853892608000, "accessible": true, "unmappedFolders": [] }
]
```

Note: `unmappedFolders` remains `[]` in this endpoint's response (cheap to return). The *count* comes from `/libraryimport/scan?rootFolderId=X&previewOnly=true`, not from this endpoint — avoids forcing a filesystem walk on every root-folder list.

### 5.4 `DELETE /api/v3/rootfolder/{id}`

- Delete row from `root_folders`.
- Before delete: count `series` where `filepath.Dir(path) == root.path`. If > 0 → 409 with body:
  ```json
  { "message": "N series still reference this root folder",
    "affectedSeries": ["Breaking Bad", "Better Call Saul", "..."] }
  ```
  (first 5 titles max).
- Does NOT delete anything on disk, ever. Just removes the DB pointer.

### 5.5 `GET /api/v3/libraryimport/scan`

File: `internal/api/v3/libraryimport.go`.

Query params:
- `rootFolderId` (required) — resolves to a root_folders row
- `previewOnly` (bool, default false) — when true, skip TVDB calls and return only the list of unmapped sub-folder names with `tvdbMatch: null`. Used by Media Management to populate the "Unmapped Folders" count column cheaply.

Behavior:
1. Load root folder; 404 if unknown.
2. Verify path still readable (`os.Stat`); 503 if not, body `{message: "root folder /data/tv is not readable"}`.
3. Verify TVDB API key configured — reads `host_config.tvdb_api_key` via the existing `hostconfig.Store.Get(ctx)` (not an environment variable; the key is set via `/settings/general` UI → `PUT /api/v3/config/general` → `host_config` table per migration 00017). If empty and `previewOnly=false`: 503 with body `{message: "TVDB API key is not configured", fixPath: "/settings/general"}`. Note: the existing `series_lookup.go` error message mentions the `SONARR2_TVDB_API_KEY` env var — that is stale copy from an earlier design and will be updated to match during implementation.
4. Walk root folder one level deep (`os.ReadDir`, not recursive), skipping dotfiles and non-directories.
5. For each sub-folder:
   - Build a lookup term: `folderName` with trailing year-parens stripped (`Breaking Bad (2008)` → `Breaking Bad`).
   - Check against in-memory index of `series.path` (built once per request from `seriesStore.List`) → flag `alreadyImported: true` and skip TVDB call.
   - If `previewOnly=true`: skip TVDB call, return with `tvdbMatch: null`.
   - Otherwise fire `series_lookup` (existing `metadatasource.MetadataSource.SearchSeries`) with concurrency cap 8 (buffered channel semaphore).
   - Take top-ranked result.
6. Return array of entries:
   ```json
   [
     {
       "folderName": "Breaking Bad (2008)",
       "relativePath": "Breaking Bad (2008)",
       "absolutePath": "/data/tv/Breaking Bad (2008)",
       "tvdbMatch": {
         "tvdbId": 81189,
         "title": "Breaking Bad",
         "year": 2008,
         "poster": "https://...",
         "overview": "..."
       },
       "alreadyImported": false
     }
   ]
   ```

Per-folder TVDB failures are **not** hard errors — the entry comes back with `tvdbMatch: null`. One weird folder name doesn't blow up the scan.

Concurrency cap is hardcoded 8 for this version; promoted to config if needed later.

### 5.6 `POST /api/v3/series` (extend existing)

The current handler (`internal/api/v3/series.go`) decodes a `seriesInput` that has no `qualityProfileId`, `seasonFolder`, or `addOptions` — it silently drops them. Library Import needs these. Extend the struct:

```go
type seriesInput struct {
    Title            string `json:"title"`
    TvdbID           int64  `json:"tvdbId"`
    Slug             string `json:"titleSlug"`
    Status           string `json:"status"`
    SeriesType       string `json:"seriesType"`
    Path             string `json:"path"`
    Monitored        bool   `json:"monitored"`
    QualityProfileID int    `json:"qualityProfileId"`    // NEW
    SeasonFolder     *bool  `json:"seasonFolder"`         // NEW — pointer so we can distinguish "not set" from false
    MonitorNewItems  string `json:"monitorNewItems"`      // NEW — "all" | "none"; default "all" if ""
    Seasons          []struct {
        SeasonNumber int  `json:"seasonNumber"`
        Monitored    bool `json:"monitored"`
    } `json:"seasons"`
    AddOptions       *addOptionsInput `json:"addOptions"` // NEW — Sonarr compat
}

type addOptionsInput struct {
    Monitor                  string `json:"monitor"`                  // see §5.7
    SearchForMissingEpisodes bool   `json:"searchForMissingEpisodes"`
    SearchForCutoffUnmetEpisodes bool `json:"searchForCutoffUnmetEpisodes"`
}
```

`seriesHandler.create` now passes `QualityProfileID`, `SeasonFolder`, and `MonitorNewItems` into `library.Series`, and consults `AddOptions.Monitor` to apply the post-create episode monitor rule (§5.7). Response is unchanged (existing `toResource` now reads from the new columns instead of hardcoding).

### 5.7 Monitor modes — post-create episode rule application

When a client (including Library Import) POSTs a new series with `addOptions.monitor`, after the series row is created and TVDB episode backfill completes, the handler iterates episodes and applies:

| Monitor mode | Rule |
|---|---|
| `all` | every episode `monitored=true` (default) |
| `future` | only episodes with `airDateUtc > now` monitored |
| `missing` | only episodes without a file monitored |
| `existing` | only episodes with a file monitored |
| `pilot` | only S01E01 monitored |
| `firstSeason` | only season 1 episodes monitored |
| `lastSeason` | only highest-numbered season monitored |
| `none` | no episodes monitored |
| `` (empty) | treated as `all` |

Implemented as a single function `library.ApplyMonitorMode(seriesID, mode) error` — testable in isolation; called from `seriesHandler.create` after episode backfill, and can be reused later by Mass Editor's bulk series monitor action.

If `addOptions.searchForMissingEpisodes=true`, after the monitor pass a `SeriesSearch` command is enqueued for the new series via the existing command dispatcher.

### 5.8 `POST /api/v3/series` batch mode? — no

The spec explicitly does **not** add a batch endpoint. Library Import calls `POST /api/v3/series` N times, sequentially (§7.3). Keeps the single path into the library, keeps partial failure debuggable.

## 6. Frontend components

### 6.1 `frontend/src/components/FileBrowserModal.tsx`

Props:
```ts
type FileBrowserModalProps = {
  isOpen: boolean
  initialPath?: string        // default '/'
  title?: string              // default 'File Browser'
  onSelect: (path: string) => void
  onCancel: () => void
}
```

Local state: `currentPath`, `children`, `loading`, `error`. Fetches `/api/v3/filesystem?path=${currentPath}` on mount and on path change.

Markup:
- Modal shell with title + close button
- Path display + clickable breadcrumb row (each segment is a `<button>` that jumps to that ancestor)
- `..` row at the top of the list when `currentPath !== '/'`
- Directory list (each a `<button>` — folder icon + name — that sets `currentPath` to its absolute path)
- Footer: **Cancel** and **OK** (OK disabled when `currentPath === '/'`)
- Keyboard: Esc → `onCancel`; Enter → `onSelect(currentPath)`; ↑/↓ cycles focused row; Enter on focused row descends

Reused by two call sites, so no behavior drift.

### 6.2 Media Management page — replace the inline text input

Current: clicking "Add Root Folder" reveals a `<input placeholder="/path/to/tv/shows" />`.

New: clicking the button opens `FileBrowserModal`. On OK → POST `/rootfolder` → toast on success / error toast on 400/409 → refresh list.

Root-folder list columns: **Path**, **Free Space** (`humanize.Bytes`), **Unmapped Folders** (count from a lightweight preview of `/libraryimport/scan?rootFolderId=X&previewOnly=true` — adding a preview mode that skips TVDB calls and only counts unmapped dirs; click routes to `/add/import?rootFolderId=X`), **Actions** (Delete with confirmation; 409 renders as a modal listing affected series).

### 6.3 `frontend/src/pages/LibraryImport.tsx` — replace the placeholder

Three local states driven by component state (no router changes):

**State A — Empty / choose folder:**
- Tips panel at top (mirrors Sonarr copy)
- Primary button **Choose folder to import from** → opens `FileBrowserModal`
- Secondary: a dropdown of existing `/rootfolder` results for re-use

**State B — Scanning:**
- Spinner with count: "Scanning /data/tv… (N folders found)"
- If 503 with `fixPath` → dismissible banner linking to the fix page

**State C — Results grid:**
Sticky top bar with bulk defaults:
- **Monitor** select: None · All Episodes · Future Episodes · Missing Episodes · Existing Episodes · Pilot Episode · First Season · Last Season
- **Quality Profile** select (from `/api/v3/qualityprofile`)
- **Series Type** select: Standard · Daily · Anime
- **Season Folder** toggle

Changing a default cascades to unchanged rows only — each row tracks a dirty flag per field.

Grid rows:
- **Checkbox** — checked by default iff `tvdbMatch !== null` && `!alreadyImported`
- **Folder Name** (left)
- **Matched Series** cell — poster + title + year; `Change` link opens a `SearchOverrideModal` backed by `/api/v3/series/lookup`; clearing marks row as "Skip"
- **Monitor / Profile / Type / Season Folder** — per-row overrides inheriting from bulk defaults until touched
- Row visual states:
  - Checked → normal
  - No match → italic grey "No match — click Change to pick one"
  - Already imported → greyed, lock icon, "Already in library", checkbox disabled

Sticky footer: **X selected · Y unmatched · Z already imported** + **Import X Series** button (disabled when X=0).

Clicking Import fires sequential POSTs to `/api/v3/series` (see §7). Per-row indicator: `pending` → `creating` (spinner) → (`created` ✓ green | `failed` ✗ red + server error + Retry). On completion, banner **"Imported X series"** linking to `/`.

### 6.4 `SearchOverrideModal` — Library Import "Change" dialog

Lives at `frontend/src/components/SearchOverrideModal.tsx`. Purpose: when a scanned folder's auto-match is wrong or missing, user clicks **Change** to search TVDB and pick a different series.

Props: `{ isOpen, initialTerm, onSelect(match: SeriesLookupResult | null), onCancel }`.
Internal: debounced input wired to existing `useSeriesLookup(term)` hook (300ms debounce). Result list renders poster + title + year + overview; click to select. Footer: "Clear match" button (calls `onSelect(null)` to mark the row "Skip") + Cancel.

### 6.5 API client — `apiFetchRaw` for v3 endpoints

Important: the default `api.{get,post,put,delete}` in `frontend/src/api/client.ts` is hardcoded to `/api/v6/` (`const API_BASE = '/api/v6'`). Every new endpoint in this spec lives under `/api/v3/` for Sonarr wire compatibility.

Pattern (see existing `ActivityBlocklist.tsx` and `SettingsTags.tsx` which already do this): new hooks use `apiFetchRaw('/api/v3/...')` and handle JSON decoding manually. Add a small helper to make this ergonomic:

```ts
// frontend/src/api/v3.ts — new file
import { apiFetchRaw, ApiError } from './client'

async function v3<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await apiFetchRaw(`/api/v3${path}`, init)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ message: res.statusText }))
    throw new ApiError(res.status, body.message ?? body.detail ?? res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const apiV3 = {
  get: <T>(path: string) => v3<T>(path),
  post: <T>(path: string, body?: unknown) => v3<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  delete: (path: string) => v3<void>(path, { method: 'DELETE' }),
}
```

Used by all new hooks in this spec. Existing `useRootFolders()` / `useAddSeries()` continue to go through the v6 default; they'll be migrated to v3 in a follow-up to keep this sub-project's blast radius minimal.

Actually — `useAddSeries` **must** migrate to v3 in this sub-project because the v6 series POST doesn't know about `qualityProfileId` either, and Library Import depends on it. That migration is scoped here.

### 6.6 New hooks

Add to `frontend/src/api/hooks.ts`:

```ts
useFilesystem(path: string)                    // GET /api/v3/filesystem?path=...
useCreateRootFolder()                          // POST /api/v3/rootfolder
useDeleteRootFolder()                          // DELETE /api/v3/rootfolder/{id}
useLibraryImportScan(rootFolderId, opts?)      // GET /api/v3/libraryimport/scan?...
useLibraryImportPreview(rootFolderId)          // same endpoint with previewOnly=true
useAddSeriesV3()                               // POST /api/v3/series — supersedes useAddSeries for Library Import call sites
```

Each follows the existing React Query pattern (`useQuery` for reads, `useMutation` + `qc.invalidateQueries` for writes).

### 6.7 Where this wires into existing code

Backend:
- New: `internal/api/v3/filesystem.go`, `internal/api/v3/libraryimport.go`
- New: `internal/rootfolder/` package (store + backfill + tests), sqlc queries in `internal/db/queries/{postgres,sqlite}/rootfolders.sql`
- Modified: `internal/api/v3/rootfolder.go` (POST, DELETE, GET rewrite)
- Modified: `internal/api/v3/series.go` (extend `seriesInput`, apply addOptions)
- Modified: `internal/library/series.go` (add `QualityProfileID`, `SeasonFolder`, `MonitorNewItems` fields)
- Modified: `internal/library/episodes.go` (new `ApplyMonitorMode` helper)
- Modified: `internal/app/app.go` (mount new routes, startup backfill, extend handler wiring to pass the new stores)
- New: migrations `00026_root_folders.sql` (both dialects) and `00027_series_library_import_fields.sql` (both dialects)

Frontend:
- New: `frontend/src/api/v3.ts` (the helper above)
- Modified: `frontend/src/api/client.ts` — no change (apiFetchRaw already exported)
- Modified: `frontend/src/api/hooks.ts` — add 6 hooks
- Modified: `frontend/src/api/types.ts` — add `FilesystemListing`, `LibraryImportEntry`, `CreateRootFolderRequest`, `AddSeriesAddOptions`; extend `AddSeriesRequest` with `seasonFolder`, `monitorNewItems`, `addOptions`
- New: `frontend/src/components/FileBrowserModal.tsx` + `.module.css` + test
- New: `frontend/src/components/SearchOverrideModal.tsx` + `.module.css` + test
- Modified: `frontend/src/pages/SettingsMediaManagement.tsx` — replace blind `<input>` + `alert()` with `FileBrowserModal` call site; upgrade root-folder table to show Free Space + Unmapped count + Delete
- Full rewrite: `frontend/src/pages/LibraryImport.tsx` + `.module.css` + test

## 7. Data flow

### 7.1 Add a root folder

1. User: Settings → Media Management → **Add Root Folder**
2. `FileBrowserModal` opens with `/` → user drills to `/data/tv` → clicks **OK**
3. Frontend `apiV3.post('/rootfolder', {path: "/data/tv"})`
4. Backend `os.Stat` → insert row → return enriched resource (201)
5. Frontend refreshes root-folder list; toast "Added /data/tv"
6. The unmapped-folder count column kicks off a parallel `apiV3.get('/libraryimport/scan?rootFolderId=X&previewOnly=true')` — lightweight, no TVDB

### 7.2 Import an existing library

1. User: Series → **Library Import**
2. Clicks **Choose folder** → `FileBrowserModal` → `/data/tv` → OK
3. Frontend checks if `/data/tv` is already in `/rootfolder`:
   - If yes: skip step 4
   - If no: POST `/api/v3/rootfolder` with that path first; on success, continue
4. Frontend `apiV3.get('/libraryimport/scan?rootFolderId=X')` → shows spinner with "Scanning /data/tv…"
5. Backend walks `/data/tv` → 50 sub-folders → issues 50 lookups (8 at a time) → returns array
6. Frontend renders grid; user tweaks bulk defaults, flips one row's Monitor, opens Change modal for two unmatched rows and picks manually
7. User clicks **Import 47 Series**
8. Frontend loops through checked rows in order. For each:
   ```json
   POST /api/v3/series
   {
     "tvdbId": 81189,
     "title": "Breaking Bad",
     "titleSlug": "breaking-bad",
     "path": "/data/tv/Breaking Bad (2008)",
     "qualityProfileId": 1,
     "seriesType": "standard",
     "seasonFolder": true,
     "monitored": true,
     "monitorNewItems": "all",
     "addOptions": { "monitor": "all", "searchForMissingEpisodes": false }
   }
   ```
   Wait for 201; advance progress; continue. On 4xx/5xx, mark row failed + continue.
9. On completion: banner "Imported 47 series, 2 failed" with list of failed rows + per-row Retry; success rows link to `/` (Series Index) once done.

### 7.3 Why the import step is sequential, not parallel

Each `POST /api/v3/series` triggers a TVDB episode-list backfill which runs through the M17 token-bucket rate limiter (5 req/s). A parallel burst of 47 creates would hit 429 and back off on nearly every one — net wall-time unchanged and the UX loses its per-row progress granularity. Sequential ~200ms/series gives the backfill breathing room, keeps per-row progress honest, and makes failures debuggable.

### 7.4 Startup data flow (first boot after this spec ships)

1. `db.Migrate(ctx, pool)` runs both 00026 and 00027 — creates `root_folders` table + adds `quality_profile_id / season_folder / monitor_new_items` to `series`
2. `rootfolder.BackfillFromSeries(ctx, rootFolderStore, seriesStore)` runs — inserts derived roots
3. HTTP server starts; new v3 routes mounted
4. On first frontend load: `useRootFolders()` now returns persisted rows rather than derived ones; behavior invisibly identical

## 8. Error handling

### 8.1 Filesystem endpoint

| Condition | Status | Body | UI response |
|---|---|---|---|
| Missing `path` | 400 | `{message: "path query parameter is required"}` | Inline toast in modal |
| `..` / outside-root | 400 | `{message: "invalid path"}` | Toast, stay on last-good path |
| Deny-listed (`/proc`, `/sys`, `/dev`) | 403 | `{message: "path is not browsable"}` | Toast |
| Not found | 404 | `{message: "path not found"}` | Toast, stay on last-good path |
| Permission denied | 403 | `{message: "permission denied"}` | Toast, stay on last-good path |
| IO error | 500 | `{message: "internal server error"}` | Toast with Retry button |

### 8.2 Root folder endpoints

| Condition | Status | Body | UI response |
|---|---|---|---|
| POST missing body / `path` field | 400 | `{message: "path is required"}` | Toast in modal |
| POST nonexistent path | 400 | `{message: "folder does not exist: /foo"}` | Toast in modal |
| POST path points to a file | 400 | `{message: "path is not a directory"}` | Toast in modal |
| POST permission denied on stat | 403 | `{message: "permission denied"}` | Toast in modal |
| POST duplicate | 409 | `{message: "root folder already exists"}` | Toast; no state change |
| DELETE unknown id | 404 | `{message: "root folder not found"}` | Silent (row already gone from list refresh) |
| DELETE with linked series | 409 | `{message: "...", affectedSeries: [...]}` | Confirm-modal describing why; lists up to 5 series titles |

### 8.3 Library import scan

| Condition | Status | Body | UI response |
|---|---|---|---|
| TVDB key missing | 503 | `{message: "TVDB API key is not configured", fixPath: "/settings/general"}` | Banner with **Configure** button routing to fix page |
| Root folder unreadable | 503 | `{message: "root folder /data/tv is not readable"}` | Banner with **Retry** button |
| Per-folder TVDB fail | (soft) | entry has `tvdbMatch: null` | Row rendered as "No match" |
| TVDB 429 during scan | (soft) | scanner waits via rate-limiter | Progress counter continues |
| Overall timeout (120s) | 504 | `{message: "scan timed out"}` | Banner with **Retry** |

### 8.4 Bulk import execution

Each POST is independent and committed locally as it lands. Failures:
- Server 4xx/5xx on a row → row marked red with server message + Retry link; batch continues
- Network blip → affected row failed with generic network error + Retry; rest continues
- User closes the tab mid-batch → successfully-imported rows stay imported (already persisted server-side)

Final banner: "Imported X succeeded, Y failed" — failed rows scrollable with per-row Retry.

### 8.5 Series POST validation (extended fields)

| Condition | Status | Body | UI response |
|---|---|---|---|
| Missing `title` or `tvdbId` | 400 | `{message: "title and tvdbId are required"}` | Row marked failed with message |
| `qualityProfileId` references non-existent profile | 400 | `{message: "qualityProfileId N does not exist"}` | Row marked failed; Library Import shows actionable hint to reload Quality Profiles |
| `addOptions.monitor` is not one of the known values | 400 | `{message: "invalid monitor mode: foo"}` | Row marked failed |
| Path already in use by another series | 409 | `{message: "series already exists at path /data/tv/Foo"}` | Row marked skipped (not failed — it's idempotent from user POV) |
| TVDB backfill failed post-create | (201 OK) | Series created, episodes empty | Row marked succeeded (series exists; episodes will fill on next RefreshSeries tick) |

## 9. Testing

### 9.1 Backend unit + integration

Location and conventions follow existing patterns in `internal/api/v3/*_test.go`.

- **`filesystem_test.go`** — creates a `t.TempDir()` fixture; lists root; asserts directories/files split; asserts `..` rejection; asserts 403 on `chmod 000` subdir; asserts symlinks resolved but traversal blocked; asserts deny-list for `/proc`, `/sys`, `/dev`.
- **`rootfolder_test.go`** — new file (existing rootfolder tests live in `task6_test.go` and cover only the old derive-from-series GET). Runs against SQLite **and** Postgres via the M1 testcontainers harness.
  - POST happy path → row persists, response enriched
  - POST nonexistent → 400
  - POST path is a file → 400 "not a directory"
  - POST duplicate → 409
  - POST missing body / missing path → 400
  - DELETE unused → 204
  - DELETE with linked series → 409 + affectedSeries populated with first 5 titles
  - DELETE unknown id → 404
  - GET returns persisted rows (post-backfill scenario)
- **`libraryimport_test.go`** — mocks `metadatasource.MetadataSource` with a fake implementation; fixture dir has:
  - 2 well-named folders that match TVDB
  - 1 unparseable name (no match → `tvdbMatch: null`)
  - 1 folder whose path already appears in `series` table (fixture) → `alreadyImported: true`
  - 1 dotfolder (skipped)
  - 1 file (skipped)
  - Asserts response shape, concurrency cap respected (counter in mock never exceeds 8), `alreadyImported` flag correct, `previewOnly=true` skips all TVDB calls.
  - Separate case: TVDB key missing → 503 with `fixPath: "/settings/general"`.
- **`rootfolder/backfill_test.go`** — seed `series` with 3 distinct `path` values whose `filepath.Dir` gives 3 distinct roots, run backfill, assert 3 rows in `root_folders`; re-run backfill, assert no duplicates (ON CONFLICT DO NOTHING works); seed with zero series, run backfill, assert no rows and no error.
- **`series_test.go` extension** — extends existing tests:
  - POST with `qualityProfileId` → row stored, resource includes it
  - POST with `seasonFolder: false` → row stored
  - POST with `addOptions.monitor: "pilot"` → S01E01 monitored, all others not (requires TVDB mock that returns episode tree)
  - POST with `addOptions.monitor: "none"` → every episode unmonitored
  - POST with `addOptions.searchForMissingEpisodes: true` → SeriesSearch command enqueued (assert via mock command dispatcher)
- **`library/monitor_mode_test.go`** — unit test the `ApplyMonitorMode` function in isolation for all 8 modes.
- **Migrations test** — run 00026 + 00027 against clean SQLite and Postgres; assert tables/columns exist; assert `INSERT ... ON CONFLICT DO NOTHING` works for `root_folders.path`; assert default values for new series columns.

### 9.2 Frontend

- **`FileBrowserModal.test.tsx`** (Vitest + React Testing Library + MSW mocking `/api/v3/filesystem`):
  - Renders with `initialPath='/data'`; list shows children
  - Click a child → `currentPath` updates, new fetch fires
  - Breadcrumb click jumps back
  - `..` row appears when `currentPath !== '/'` and ascends one level
  - Esc fires `onCancel`; Enter fires `onSelect(currentPath)`
  - OK disabled at `/`
  - Arrow keys move focus through list
  - 403 response shows toast, modal stays open on last-good path
- **`SearchOverrideModal.test.tsx`**:
  - Renders with initialTerm, debounced input fires after 300ms
  - Results list renders correctly
  - Click result fires `onSelect(result)`
  - Clear match button fires `onSelect(null)`
  - Cancel fires `onCancel`
- **`LibraryImport.test.tsx`** (MSW for all endpoints):
  - Empty state renders; clicking Choose Folder opens `FileBrowserModal`
  - Selecting a folder that's already a root folder skips POST, just fires scan
  - Selecting a folder that's not a root folder POSTs to `/rootfolder` first, then scan
  - Grid renders 3 row types (matched / unmatched / already-imported) with correct default check states (matched=checked, unmatched=unchecked, already=disabled)
  - Bulk default change (e.g. change Quality Profile) cascades to non-dirty rows only — set one row's QP explicitly, change bulk QP, assert touched row kept its override
  - Change button on unmatched row opens `SearchOverrideModal`; selecting a result updates the row's tvdbMatch and checkbox becomes checked
  - Import button disabled when 0 rows selected
  - Sequential import: mock server where 2nd row returns 500 and 1st/3rd return 201; verify POSTs happen in order (assert Nth request only after N-1th resolves), per-row progress states advance, final banner reports 2 succeeded / 1 failed
  - Retry on failed row re-fires POST and transitions to succeeded on 201
  - TVDB-missing banner shows when scan returns 503 with `fixPath`; clicking Configure navigates
  - `alreadyImported` rows have checkbox disabled and lock icon
- **`SettingsMediaManagement.test.tsx`**:
  - Clicking Add Root Folder opens `FileBrowserModal` (not the old alert-prompt)
  - On OK, POST fires and list refreshes
  - Delete with no linked series → removal succeeds
  - Delete with linked series → 409 response renders confirm-modal listing affected series

Existing tests for `SettingsMediaManagement` that rely on the current alert-based flow must be removed or rewritten.

### 9.3 Manual smoke on real sonarr2 instance

After the change is deployed:
- Create `/tmp/fixtures/tv/Breaking Bad (2008)` and `/tmp/fixtures/tv/Better Call Saul`
- Settings → Media Management → Add Root Folder → pick `/tmp/fixtures/tv` → verify row appears with free-space
- Series → Library Import → scan `/tmp/fixtures/tv` → verify both folders show matched → Import 2 Series → verify they appear on Series Index and are persisted after a restart

### 9.4 CI

All new tests run under the existing `make lint && make test` pipeline. No new dependencies.

## 10. Dependencies / prerequisites

- User must configure a TVDB API key. The key lives in `host_config.tvdb_api_key` (migration 00017) and is set via Settings → General → TVDB API Key field (already wired via `useUpdateGeneralSettings` → `PUT /api/v3/config/general`). The Library Import scanner reads it via `hostconfig.Store.Get(ctx)`. Scanner fails-loud with a linked fix-path banner when the key is missing.
- Existing `series_lookup.go` emits a stale error message referencing `SONARR2_TVDB_API_KEY` env var; implementation updates this to reference the Settings page instead.
- Docker-deployed instances need the filesystem they want to import from mounted into the container; the `/api/v3/filesystem` endpoint only sees what the process sees. Documented in a README update that accompanies the implementation (per the `feedback_readme_updates.md` memory).
- `library.Series` and its sqlc queries gain three fields (`quality_profile_id`, `season_folder`, `monitor_new_items`). 17 files construct `library.Series{...}` literals today (grep: `library\.Series\{` across the codebase — includes handlers, migration tool, RSS sync, refresh commands, scan commands, import pipeline, app bootstrap, and tests). All become compile-safe because the new fields have sensible zero values (`nil`, `false`, `""`) and the migration 00027 backfills existing rows with `quality_profile_id = 1`. Still, as part of implementation: scan every `library.Series{...}` literal to confirm no caller relied on the older implicit behavior.
- Migration-tool import path (`internal/migrate/migrate.go`, ingests from an existing Sonarr v3/v4 SQLite DB per M21) — when reading Sonarr rows, start populating the `qualityProfileId` field rather than dropping it. This is a migration-tool fidelity improvement that becomes possible once the column exists.

## 11. Risks

- **`/api/v3/filesystem` is a sensitive endpoint.** An attacker who authenticates can enumerate the container's filesystem. Mitigated by the existing API-key auth; the deny-list on `/proc`, `/sys`, `/dev` limits worst-case info leak. Not more exposure than Sonarr has with the same endpoint.
- **Folder-name to TVDB matching is fragile for non-English titles / ambiguous names.** Accepted — the Change override is the escape hatch. Surface the problem clearly by showing "No match" rather than guessing wrong.
- **Large libraries (hundreds of sub-folders) make the scan endpoint slow.** The 120s timeout bounds worst case. If it's a real problem we can move to SSE-streamed results in a follow-up; not needed for v1.
- **Concurrent mutation during scan** (user adds a series while scanning the same folder) could produce a racy `alreadyImported` flag. Accepted — worst case is the same folder appears once as importable and once as imported on the next refresh.
- **v6 rootfolder.go stays unchanged** and continues to derive folders from series paths. Any v6 client will see inconsistent data vs v3 until v6 is updated in a follow-up sub-project. Internal frontend callers get migrated to v3 in this sub-project, so user-visible impact is zero.
- **`seasonFolder` stored as BOOLEAN default TRUE for all existing series** — pre-existing rows will get `true`. If a user has series where they manually disabled season folders (which today isn't possible because the column didn't exist), they'd need to re-set the flag. Acceptable: Sonarr's default is true and this is a brand-new field.
- **Migration 00027 on large series tables** — `ALTER TABLE series ADD COLUMN ... DEFAULT ...` is fast on SQLite (column-level default, no rewrite) but in Postgres pre-11 would rewrite the table. Postgres 11+ stores the default in metadata — fine. We assume Postgres 11+ (consistent with goose/pgx requirements). Noted in the migration file comment.
- **Migration 00027 backfills `quality_profile_id=NULL` for existing rows**. Every existing row has a functioning quality profile derived elsewhere (hardcoded `1` in `toResource`). After this migration, existing series will show `qualityProfileId: null` in API responses until a user edits them, or a follow-up backfill sets them to `1`. Decision: include a `UPDATE series SET quality_profile_id = 1 WHERE quality_profile_id IS NULL` in migration 00027 to preserve current behavior exactly.

## 12. Deliverables checklist

### Backend

Migrations:
- [ ] `internal/db/migrations/postgres/00026_root_folders.sql`
- [ ] `internal/db/migrations/sqlite/00026_root_folders.sql`
- [ ] `internal/db/migrations/postgres/00027_series_library_import_fields.sql` (adds `quality_profile_id`, `season_folder`, `monitor_new_items` to `series`; backfills `quality_profile_id = 1`)
- [ ] `internal/db/migrations/sqlite/00027_series_library_import_fields.sql`

sqlc queries + generated code:
- [ ] `internal/db/queries/postgres/root_folders.sql` + generated `internal/db/gen/postgres/root_folders.sql.go`
- [ ] `internal/db/queries/sqlite/root_folders.sql` + generated `internal/db/gen/sqlite/root_folders.sql.go`
- [ ] Update `internal/db/queries/{postgres,sqlite}/series.sql` for new columns + regen

Domain:
- [ ] New package `internal/rootfolder/` with `Store` interface + postgres/sqlite impls + `BackfillFromSeries` function + tests
- [ ] Extend `internal/library/series.go` `Series` struct with `QualityProfileID`, `SeasonFolder`, `MonitorNewItems`
- [ ] Extend `internal/library/series.go` `SeriesStore` implementations to read/write the new columns
- [ ] New `internal/library/monitor_mode.go` with `ApplyMonitorMode(ctx, seriesID, mode) error` + tests

Handlers:
- [ ] New `internal/api/v3/filesystem.go` + `filesystem_test.go`
- [ ] New `internal/api/v3/libraryimport.go` + `libraryimport_test.go`
- [ ] Rewrite `internal/api/v3/rootfolder.go` (POST, DELETE, GET from `root_folders` table) + extend `rootfolder_test.go`
- [ ] Extend `internal/api/v3/series.go` `seriesInput` with `qualityProfileId`, `seasonFolder`, `monitorNewItems`, `addOptions` + update `create` handler + extend `series_test.go`
- [ ] Fix stale TVDB-key error message in `internal/api/v3/series_lookup.go` (and `v6/series_lookup.go` mirror) to point at `/settings/general`

Wiring:
- [ ] `internal/app/app.go`:
  - Construct `rootfolder.Store` and pass it to the new handlers + Library Import handler
  - Mount `MountFilesystem`, update `MountRootFolder`, `MountLibraryImport`
  - After `db.Migrate`, call `rootfolder.BackfillFromSeries`

### Frontend

API layer:
- [ ] New `frontend/src/api/v3.ts` — `apiV3` helper wrapping `apiFetchRaw('/api/v3/...')`
- [ ] Extend `frontend/src/api/types.ts`: `FilesystemListing`, `FilesystemEntry`, `LibraryImportEntry`, `CreateRootFolderRequest`, `AddSeriesAddOptions`; extend `AddSeriesRequest` with `seasonFolder`, `monitorNewItems`, `addOptions`
- [ ] Extend `frontend/src/api/hooks.ts` with: `useFilesystem`, `useCreateRootFolder`, `useDeleteRootFolder`, `useLibraryImportScan`, `useLibraryImportPreview`, `useAddSeriesV3`; deprecate (keep for 1 release) the v6-bound `useRootFolders` + `useAddSeries` — migrate Settings/MediaManagement, Library Import, and AddSeries pages to the v3 hooks

Components:
- [ ] New `frontend/src/components/FileBrowserModal.tsx` + `.module.css` + `FileBrowserModal.test.tsx`
- [ ] New `frontend/src/components/SearchOverrideModal.tsx` + `.module.css` + `SearchOverrideModal.test.tsx`

Pages:
- [ ] Modify `frontend/src/pages/SettingsMediaManagement.tsx` — replace alert-based root-folder add with `FileBrowserModal`; upgrade root-folder table to show Path / Free Space / Unmapped Folders count / Delete; handle 409 with affectedSeries modal + `SettingsMediaManagement.test.tsx` updates
- [ ] Modify `frontend/src/pages/AddSeries.tsx` — switch from `useAddSeries` (v6) to `useAddSeriesV3`; wire up the `qualityProfileId` / `seasonFolder` / `monitorNewItems` / `addOptions` fields that the Add form already collects but can't currently send; add the missing controls if the Add form doesn't expose them yet
- [ ] Rewrite `frontend/src/pages/LibraryImport.tsx` + new `LibraryImport.module.css` + `LibraryImport.test.tsx`

### Docs

- [ ] README section on root-folder Docker-mount requirement and how Library Import works
- [ ] Update `docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md` status table: Library Import moves from "not started" to "done"; Add New Series add-form becomes "partial" pending its own sub-project
- [ ] Implementation plan at `docs/superpowers/plans/2026-04-14-root-folders-library-import.md` (produced by `writing-plans` skill in the next step)

### CI / verification

- [ ] `make lint && make test` passes
- [ ] Manual smoke per §9.3 on the user's sonarr2 instance
- [ ] Push + watch CI per the `feedback_push_and_watch_ci.md` memory

## 13. Revision log

**2026-04-14 (initial):** Written after design brainstorm.

**2026-04-14 (revision, post-codebase-review):** Material additions after auditing actual sonarr2 source:
- Added §4.2 — `series` table needs new columns (`quality_profile_id`, `season_folder`, `monitor_new_items`). `library.Series` struct was missing these; the existing v3 `seriesInput` silently dropped them. Library Import depends on them.
- Added §5.7 — `ApplyMonitorMode` helper for the 8 Sonarr monitor modes; called post-create.
- Added §5.8 — explicit decision to not introduce a batch endpoint.
- Added §6.4 — `SearchOverrideModal` component (was implicit in §6.3 but deserved its own spec).
- Added §6.5 — `apiV3` helper. Discovered the default frontend `api` object goes to `/api/v6/` (hardcoded `API_BASE`), so new v3 endpoints need their own helper and existing `useAddSeries` needs migration.
- Added §6.6 — explicit new-hook list (6 hooks).
- Fixed §4.3 — the backfill is at app startup, not a goose migration hook. Existing migrations are pure-SQL (goose/v3); don't diverge from the pattern.
- Fixed §5.5 — TVDB key source is `host_config.tvdb_api_key` (migration 00017), accessed via `hostconfig.Store`. Existing `series_lookup.go` error message pointing at the env var is stale and will be updated.
- Fixed §10 — clarified the TVDB key plumbing path.
- Added §11 — new risks: v6 staleness, `seasonFolder` default, ALTER TABLE on large Postgres series tables, `quality_profile_id=NULL` backfill (decision: migration 00027 sets it to 1 for existing rows).
- Expanded §12 — deliverables checklist now enumerates every file touched.
