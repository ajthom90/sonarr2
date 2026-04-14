# Connect + Import Lists + Metadata Settings Pages — Design

**Date:** 2026-04-14
**Status:** Approved.
**Sub-project:** #3 of the post-audit Sonarr-parity UI program.
**Depends on:** Sub-project #2 (shared `ProviderListSection`, `ProviderPickerModal`, `ProviderSettingsModal`, `SchemaFormField`).

## Goal

Replace three Settings placeholder pages — `/settings/connect`, `/settings/importlists`, `/settings/metadata` — with working UIs that reuse the provider-pattern components shipped in sub-project #2.

## Scope (honest)

Backend reality:
- **`/api/v3/notification`** — full CRUD (backend wired). Resource shape includes `onGrab`, `onDownload`, `onHealthIssue`, `tags`, standard provider fields.
- **`/api/v3/importlist`** — **read-only**: GET `/schema` and GET `/` (always returns `[]`). No store, no POST/PUT/DELETE. Migration 00023 created `import_lists` + `import_list_exclusions` tables but no Go store or sqlc queries consume them yet.
- **`/api/v3/metadata`** — **read-only**: GET `/schema` and GET `/` (always returns `[]`). No store at all.

So this sub-project builds:

1. **`/settings/connect`** — full CRUD with the same pattern as Indexers/Download Clients. `NotificationExtras = { onGrab, onDownload, onHealthIssue, tags }`. 3 event-trigger checkboxes (what backend currently supports) + tag picker (read-only today — no tag selector widget exists; surface a comma-separated integer-id input as a placeholder).
2. **`/settings/importlists`** — **browsable read-only catalog**. Lists the 11 registered providers from `/schema`, each rendered as a card with "Add Instance — coming in follow-up". No Add button, no picker, no save. The page does tell users which providers sonarr2 supports so they know what's coming.
3. **`/settings/metadata`** — same pattern as Import Lists. Shows the 4 metadata consumers (Kodi, Plex, Roksbox, WDTV). Read-only catalog.

## Non-goals

- **Persisted CRUD for Import Lists** — backend needs `internal/importlist/sqlite.go` + `postgres.go` + sqlc queries + handler extensions. Separate sub-project.
- **Persisted CRUD for Metadata** — needs a new migration (table doesn't exist yet), new package, new handler. Separate sub-project.
- **Extending notification backend to cover the 13 Sonarr event triggers** (OnRename, OnSeriesAdd, OnSeriesDelete, OnEpisodeFileDelete, OnEpisodeFileDeleteForUpgrade, OnHealthRestored, OnApplicationUpdate, OnManualInteractionRequired, IncludeHealthWarnings). Backend only has 3 today; expand in a follow-up.
- **Tag selector widget** — render as raw integer-id list for now. A proper multiselect tag picker is a separate UI component.
- **Options subsection on Import Lists** (Clean Library Level) — depends on persisted instances, which aren't yet supported.

## New wire types (append to `frontend/src/api/types.ts`)

```ts
export interface NotificationResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  onGrab: boolean
  onDownload: boolean
  onHealthIssue: boolean
  tags: number[]
  added?: string
}
```

`ProviderSchema` (from sub-project #2) is already a perfect fit for `/importlist/schema` and `/metadata/schema`.

## New hooks (extend `frontend/src/api/hooks.ts`)

```ts
useNotificationSchema()        // GET /api/v3/notification/schema
useNotifications()             // GET /api/v3/notification
useCreateNotification()
useUpdateNotification()
useDeleteNotification()

useImportListSchema()          // GET /api/v3/importlist/schema
useMetadataSchema()            // GET /api/v3/metadata/schema
```

(Read-only for Import Lists + Metadata — no create/update/delete hooks yet.)

## Pages

### `/settings/connect`

Mirror `/settings/indexers` / `/settings/downloadclients`. `NotificationExtras`:
```ts
interface NotificationExtras {
  onGrab: boolean
  onDownload: boolean
  onHealthIssue: boolean
  tags: number[]
}
```

Extras renderer draws 3 event checkboxes + a "Tags" input treated as comma-separated integer ids (trim, split, filter NaN). Card shows the three On* flags as green/grey pills.

### `/settings/importlists` (read-only)

Reuses `ProviderListSection<T>` with `items = schemas` (not instances). `renderCard` draws the provider name + implementation + a muted "Not yet configurable" tag. `onAdd` is not wired; the `+ Add` button is replaced with a banner: *"Import List persistence is coming in a follow-up sub-project. The 11 providers below are supported server-side but can't be saved yet."*

### `/settings/metadata` (read-only)

Same pattern. Banner: *"Metadata consumer persistence is coming in a follow-up sub-project. The 4 consumers below (Kodi/Emby, Plex, Roksbox, WDTV) are supported server-side but can't be saved yet."*

## Deliverables

- [ ] `frontend/src/api/types.ts` — add `NotificationResource`
- [ ] `frontend/src/api/hooks.ts` — add 7 hooks
- [ ] `frontend/src/pages/SettingsConnect.tsx` + `.module.css` — full rewrite
- [ ] `frontend/src/pages/SettingsImportLists.tsx` + `.module.css` — rewrite as read-only catalog
- [ ] `frontend/src/pages/SettingsMetadata.tsx` + `.module.css` — rewrite as read-only catalog
- [ ] README bullet
- [ ] M24 status update
- [ ] Rebuild frontend + push + watch CI

## Verification

- `npm run build` clean after each task
- Manual smoke (post-deploy): add a Discord notification in Connect, save, edit, delete. Browse Import Lists + Metadata catalogs.

## Follow-ups

- SP #4 candidate: backend persistence for Import Lists + Metadata + full Connect event triggers.
- Tag selector widget (reusable component once real tags UX is needed).
