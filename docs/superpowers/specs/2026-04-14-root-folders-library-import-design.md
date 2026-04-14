# Root Folders + Library Import — Design

**Date:** 2026-04-14
**Status:** Approved (pending implementation-plan hand-off)
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

New migration `internal/db/migrations/{postgres,sqlite}/00026_root_folders.sql`:

```sql
CREATE TABLE root_folders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_root_folders_path ON root_folders(path);
```

(Postgres version uses `SERIAL` instead of `INTEGER PRIMARY KEY AUTOINCREMENT` and `TIMESTAMPTZ`.)

Deliberately minimal. `accessible` and `free_space` are computed on read via `os.Stat` and `syscall.Statfs_t`; a mount that disappears between calls is visible immediately instead of showing stale persisted state.

**Backfill on first run of migration:** a Go-side migration hook (not SQL) reads distinct `filepath.Dir(series.path)` from the `series` table and inserts any not already present in `root_folders`. Installations that pre-date this spec keep working without user intervention. Wrapped in `INSERT ... ON CONFLICT DO NOTHING` so re-runs are safe.

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

Request: `{ "path": "/data/tv" }`

Behavior:
- `os.Stat` the path; if `!IsDir` or error → 400 with specific message
- Insert into `root_folders`; on unique-constraint violation → 409 `{message: "root folder already exists"}`
- Return the populated `rootFolderResource` (id, path, freeSpace via `statfs`, accessible=true, unmappedFolders=[])

### 5.3 `GET /api/v3/rootfolder` (upgrade)

Today: infers root folders from distinct `filepath.Dir(series.path)`.
After: merges persisted rows from `root_folders` with derived rows (for series rooted at paths not yet persisted), dedup by path, persisted rows win.

`freeSpace` and `accessible` computed per row on each call. A slow mount would slow this response; acceptable for a settings page (user is there infrequently) and cacheable by frontend.

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
2. Verify path still readable; 503 if not, body `{message: "root folder /data/tv is not readable"}`.
3. Verify TVDB API key configured; 503 if not, body `{message: "TVDB API key is not configured", fixPath: "/settings/general"}`.
4. Walk root folder one level deep (`os.ReadDir`, not recursive), skipping dotfiles and non-directories.
5. For each sub-folder:
   - Build a lookup term: `folderName` with trailing year-parens stripped (`Breaking Bad (2008)` → `Breaking Bad`).
   - Check against in-memory index of `series.path` → flag `alreadyImported: true` and skip TVDB call.
   - Otherwise fire `series_lookup` (existing TVDB client) with concurrency cap 8 (semaphore).
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

### 5.6 `POST /api/v3/series` (existing — no change)

Library Import reuses this one series at a time, sequentially. Rationale in §7.

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

### 6.4 Where this wires into existing code

- `frontend/src/api/client.ts` — add `filesystem`, `rootFolder.{create,remove}`, `libraryImport.scan` functions
- `frontend/src/api/hooks.ts` — add React Query hooks matching the existing pattern
- `frontend/src/api/types.ts` — add response types
- `frontend/src/pages/LibraryImport.tsx` — rewrite (currently 14 lines)
- `frontend/src/pages/SettingsMediaManagement.tsx` — add the `FileBrowserModal` call site
- `frontend/src/components/FileBrowserModal.tsx` + `.module.css` — new

## 7. Data flow

### 7.1 Add a root folder

1. User: Settings → Media Management → **Add Root Folder**
2. `FileBrowserModal` opens with `/` → user drills to `/data/tv` → clicks **OK**
3. Frontend POST `/api/v3/rootfolder` `{path: "/data/tv"}`
4. Backend `os.Stat` → insert row → return enriched resource
5. Frontend refreshes root-folder list; toast "Added /data/tv"

### 7.2 Import an existing library

1. User: Series → **Library Import**
2. Clicks **Choose folder** → modal → `/data/tv` → OK
3. Frontend detects `/data/tv` isn't in `/rootfolder` yet → POST first, then move on
4. Frontend `GET /libraryimport/scan?rootFolderId=X` → shows spinner
5. Backend walks `/data/tv` → 50 sub-folders → issues 50 lookups (8 at a time) → returns array
6. Frontend renders grid; user tweaks bulk defaults, flips one row's Monitor, overrides two unmatched rows
7. User clicks **Import 47 Series**
8. Frontend loops through checked rows in order: for each, POST `/api/v3/series` with `{tvdbId, path, monitored, qualityProfileId, seriesType, seasonFolder}`; wait for response; advance progress; continue
9. On completion: banner "Imported 47 series" → Series Index

### 7.3 Why the import step is sequential, not parallel

Each `POST /api/v3/series` triggers a TVDB episode-list backfill which runs through the M17 token-bucket rate limiter (5 req/s). A parallel burst of 47 creates would hit 429 and back off on nearly every one — net wall-time unchanged and the UX loses its per-row progress granularity. Sequential ~200ms/series gives the backfill breathing room, keeps per-row progress honest, and makes failures debuggable.

## 8. Error handling

### 8.1 Filesystem endpoint

| Condition | Status | Body | UI response |
|---|---|---|---|
| Missing `path` | 400 | `{message: "path query parameter is required"}` | Inline toast in modal |
| `..` / outside-root | 400 | `{message: "invalid path"}` | Toast, stay on last-good path |
| Not found | 404 | `{message: "path not found"}` | Toast, stay on last-good path |
| Permission denied | 403 | `{message: "permission denied"}` | Toast, stay on last-good path |
| IO error | 500 | `{message: "internal server error"}` | Toast with Retry button |

### 8.2 Root folder endpoints

| Condition | Status | Body | UI response |
|---|---|---|---|
| POST nonexistent path | 400 | `{message: "folder does not exist: /foo"}` | Toast in modal |
| POST duplicate | 409 | `{message: "root folder already exists"}` | Toast; no state change |
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

## 9. Testing

### 9.1 Backend unit + integration

Location and conventions follow existing patterns in `internal/api/v3/*_test.go`.

- **`filesystem_test.go`** — creates a `t.TempDir()` fixture; lists root; asserts directories/files split; asserts `..` rejection; asserts 403 on `chmod 000` subdir; asserts symlinks resolved but traversal blocked.
- **`rootfolder_test.go`** — extends existing. Runs against SQLite **and** Postgres via the M1 testcontainers harness.
  - POST happy path → row persists, response enriched
  - POST nonexistent → 400
  - POST duplicate → 409
  - DELETE unused → 204
  - DELETE with linked series → 409 + affectedSeries populated
  - GET merges persisted + derived, dedup by path
- **`libraryimport_test.go`** — mocks `series_lookup.Client` with a fake interface implementation; fixture dir has:
  - 2 well-named folders that match TVDB
  - 1 unparseable name
  - 1 folder whose path already appears in `series` table (fixture)
  - 1 dotfolder (skipped)
  - Asserts response shape, concurrency cap respected (via counter in mock), `alreadyImported` flag correct.
- **Migration 00026 backfill test** — seed `series` with 3 distinct root paths, run migration, assert 3 rows in `root_folders`; re-run migration, assert no duplicates.

### 9.2 Frontend

- **`FileBrowserModal.test.tsx`** (Vitest + React Testing Library + MSW mocking `/api/v3/filesystem`):
  - Renders with `initialPath='/data'`; list shows children
  - Click a child → `currentPath` updates, new fetch fires
  - Breadcrumb click jumps back
  - Esc fires `onCancel`; Enter fires `onSelect(currentPath)`
  - OK disabled at `/`
- **`LibraryImport.test.tsx`** (MSW for all endpoints):
  - Empty state renders; clicking Choose Folder opens modal
  - Selecting a folder fires scan
  - Grid renders 3 row types (matched / unmatched / already-imported) with correct default check states
  - Bulk default change cascades to non-dirty rows only (set one row dirty, change default, assert dirty row didn't move)
  - Import button disabled when 0 selected
  - Sequential import shows per-row progress; 2nd row fails, 1st and 3rd succeed; final banner reports 2 succeeded / 1 failed; Retry on failed row re-fires POST

### 9.3 Manual smoke on real sonarr2 instance

After the change is deployed:
- Create `/tmp/fixtures/tv/Breaking Bad (2008)` and `/tmp/fixtures/tv/Better Call Saul`
- Settings → Media Management → Add Root Folder → pick `/tmp/fixtures/tv` → verify row appears with free-space
- Series → Library Import → scan `/tmp/fixtures/tv` → verify both folders show matched → Import 2 Series → verify they appear on Series Index and are persisted after a restart

### 9.4 CI

All new tests run under the existing `make lint && make test` pipeline. No new dependencies.

## 10. Dependencies / prerequisites

- User must configure a TVDB API key via `/settings/general` (already wired) before Library Import will match anything. The scanner fails-loud with a link to that page when the key is missing.
- Docker-deployed instances need the filesystem they want to import from mounted into the container; the `filesystem` endpoint only sees what the process sees. Documented in a README update that accompanies the implementation (per the `feedback_readme_updates.md` memory).

## 11. Risks

- **`/api/v3/filesystem` is a sensitive endpoint.** An attacker who authenticates can enumerate the container's filesystem. Mitigated by the existing API-key auth; the deny-list on `/proc`, `/sys`, `/dev` limits worst-case info leak. Not more exposure than Sonarr has with the same endpoint.
- **Folder-name to TVDB matching is fragile for non-English titles / ambiguous names.** Accepted — the Change override is the escape hatch. Surface the problem clearly by showing "No match" rather than guessing wrong.
- **Large libraries (hundreds of sub-folders) make the scan endpoint slow.** The 120s timeout bounds worst case. If it's a real problem we can move to SSE-streamed results in a follow-up; not needed for v1.
- **Concurrent mutation during scan** (user adds a series while scanning the same folder) could produce a racy `alreadyImported` flag. Accepted — worst case is the same folder appears once as importable and once as imported on the next refresh.

## 12. Deliverables checklist

Backend:
- [ ] Migration `00026_root_folders.sql` (postgres + sqlite) + Go-side backfill
- [ ] sqlc queries + generated code
- [ ] `internal/api/v3/filesystem.go` + test
- [ ] `internal/api/v3/rootfolder.go` extensions + test
- [ ] `internal/api/v3/libraryimport.go` + test
- [ ] `internal/app/app.go` wiring of new handlers

Frontend:
- [ ] `frontend/src/api/{client,hooks,types}.ts` extensions
- [ ] `frontend/src/components/FileBrowserModal.tsx` + `.module.css` + test
- [ ] `frontend/src/pages/SettingsMediaManagement.tsx` — replace blind text input
- [ ] `frontend/src/pages/LibraryImport.tsx` + `.module.css` — full rewrite + test
- [ ] `frontend/src/pages/LibraryImport.module.css`

Docs:
- [ ] README section on root-folder docker-mount requirement
- [ ] Update M24 design doc status table: Library Import moves from "not started" to "done"
