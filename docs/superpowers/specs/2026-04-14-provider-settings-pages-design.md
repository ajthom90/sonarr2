# Provider Settings Pages (Indexers + Download Clients) — Design

**Date:** 2026-04-14
**Status:** Approved (scope + approach + section 1 approved in-session; sections 2+3 included here).
**Sub-project:** #2 of the post-audit Sonarr-parity UI program.
**Depends on:** Sub-project #1 (`apiV3` helper from T12, already shipped on `feature/root-folders-library-import`).

## 1. Goal

Replace the two placeholder pages `/settings/indexers` and `/settings/downloadclients` with full Sonarr-parity UIs backed by the existing v3 provider CRUD + `/schema` endpoints. Users can add, edit, and remove indexers / download clients using schema-driven forms that work for every registered provider (10 indexer + 20 download-client implementations).

## 2. Non-goals

- **Test All / per-row Test actions.** Sonarr has these; sonarr2 backend has no `/api/v3/indexer/test` or `/api/v3/downloadclient/test` endpoints yet. Deferred to its own sub-project.
- **Indexers → Options section** (Minimum Age, Retention, Maximum Size, RSS Sync Interval). Likely lives in `host_config` or a sibling table; this spec doesn't touch it.
- **Connect / Import Lists / Metadata settings pages.** Same provider-picker pattern but separate pages. The shared components built here will land those as fast follow-ups (#2B, #2C, #2D).
- **Backend changes.** Everything we need is already exposed by the v3 API.

## 3. Architecture overview

```
┌──────────────────────────────────────────────────────────┐
│                     Frontend                              │
│                                                           │
│  SettingsIndexers.tsx      SettingsDownloadClients.tsx    │
│  (thin wrapper, ~80 LOC)    (thin wrapper, ~80 LOC        │
│                              + RemotePathMappings panel)  │
│        │                           │                      │
│        ▼                           ▼                      │
│    ┌─────────────────────────────────────┐                │
│    │ ProviderListSection<T>              │ — generic      │
│    │   ProviderPickerModal               │   shared       │
│    │   ProviderSettingsModal             │                │
│    │   SchemaFormField                   │                │
│    └─────────────────────────────────────┘                │
│                   │                                       │
│                   ▼                                       │
│              api/v3 hooks                                 │
└──────────────────────────────────────────────────────────┘
                    │
   GET /schema, GET /, POST /, PUT /{id}, DELETE /{id}
   on /api/v3/indexer and /api/v3/downloadclient
   + GET/POST/DELETE /api/v3/remotepathmapping
```

## 4. Types (extend `frontend/src/api/types.ts`)

```ts
export interface ProviderFieldSchema {
  name: string
  label: string
  type: 'text' | 'password' | 'number' | 'checkbox' | 'select' | 'multiselect'
  required?: boolean
  default?: string
  placeholder?: string
  helpText?: string
  advanced?: boolean
}

export interface ProviderSchema {
  implementation: string
  name: string
  fields: ProviderFieldSchema[]
}

export interface IndexerResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  enableRss: boolean
  enableAutomaticSearch: boolean
  enableInteractiveSearch: boolean
  priority: number
  added?: string
}

export interface DownloadClientResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  enable: boolean
  priority: number
  added?: string
}

export interface RemotePathMapping {
  id: number
  host: string
  remotePath: string
  localPath: string
}
```

Backend wire note: the `/api/v3/indexer` resource has `fields` as `json.RawMessage` (serialized object). On the frontend we deserialize at fetch time and re-serialize at mutation time — handled in the hook layer.

## 5. Hooks (extend `frontend/src/api/hooks.ts`)

Using `apiV3` from T12:

```ts
useIndexerSchema()            // GET /api/v3/indexer/schema  → ProviderSchema[]
useIndexersV3()               // GET /api/v3/indexer → IndexerResource[]
useCreateIndexer()            // POST
useUpdateIndexer()            // PUT /api/v3/indexer/{id}
useDeleteIndexerV3()          // DELETE /api/v3/indexer/{id}

useDownloadClientSchema()     // same shape, /downloadclient
useDownloadClientsV3()
useCreateDownloadClient()
useUpdateDownloadClient()
useDeleteDownloadClientV3()

useRemotePathMappings()        // GET /api/v3/remotepathmapping
useCreateRemotePathMapping()   // POST
useDeleteRemotePathMapping()   // DELETE /{id}
```

All follow existing React Query patterns — `useQuery` for reads, `useMutation` + `qc.invalidateQueries` for writes.

## 6. Shared components

### 6.1 `frontend/src/components/SchemaFormField.tsx`

Single-field renderer. Props:
```ts
interface SchemaFormFieldProps {
  schema: ProviderFieldSchema
  value: unknown
  onChange: (next: unknown) => void
}
```

One switch statement over `schema.type`:
- **`text`** → `<input type="text">`
- **`password`** → `<input type="password">`
- **`number`** → `<input type="number">` (stores as number, not string)
- **`checkbox`** → `<input type="checkbox">`
- **`select`** / **`multiselect`** → falls back to `<input type="text">` for v1 because the backend's current `FieldSchema` doesn't emit options. Add a TODO comment pointing at the backend gap.

Renders `label`, `helpText` below, `required` asterisk after label. Uses the same dark-theme CSS variables as T13/T14 modals.

### 6.2 `frontend/src/components/ProviderSettingsModal.tsx`

Schema-driven form modal. Props:
```ts
interface ProviderSettingsModalProps<Extras> {
  isOpen: boolean
  title: string                                    // e.g. "Add Newznab Indexer"
  schema: ProviderSchema                           // fields to render
  initialValues: Record<string, unknown>           // empty or from edit row
  initialName: string                              // row.name or schema.name
  extras: Extras                                   // indexer-specific / dl-client-specific toggles
  renderExtras: (extras: Extras, set: (e: Extras) => void) => ReactNode
  onSubmit: (next: { name: string; fields: Record<string, unknown>; extras: Extras }) => void
  onCancel: () => void
}
```

`extras` is generic because the two pages need different extra fields:
- Indexer: `{ enableRss, enableAutomaticSearch, enableInteractiveSearch, priority }`
- DL client: `{ enable, priority }`

Each page provides a `renderExtras` callback that draws checkboxes for its toggles.

Applies `schema.field.default` as initial when `initialValues[field.name]` is undefined. Advanced fields hidden under a "Show Advanced" toggle (same pattern as Sonarr).

Validation: required fields must be non-empty. Disable Save if any required field is empty.

### 6.3 `frontend/src/components/ProviderPickerModal.tsx`

Lists all available provider schemas. Props:
```ts
interface ProviderPickerModalProps {
  isOpen: boolean
  providers: ProviderSchema[]
  onPick: (schema: ProviderSchema) => void
  onCancel: () => void
}
```

Grid of cards — each card shows `schema.name` (e.g. "Newznab") + `schema.implementation` in muted text. Click → `onPick(schema)` → host page replaces picker with `ProviderSettingsModal`.

### 6.4 `frontend/src/components/ProviderListSection.tsx`

Generic list + add-button. Props:
```ts
interface ProviderListSectionProps<T> {
  title: string
  items: T[]
  renderCard: (item: T) => ReactNode           // each page builds its own card
  onAdd: () => void                             // opens picker
  onEdit: (item: T) => void
  onDelete: (item: T) => void
}
```

Renders an `<h2>{title}</h2>`, the card grid, and a bottom-right `Add` button. Each card comes from `renderCard` which the consuming page provides — the page owns how toggles + priority are drawn.

## 7. Page wrappers

### 7.1 `frontend/src/pages/SettingsIndexers.tsx`

Full rewrite. ~80 lines:

```tsx
export function SettingsIndexers() {
  const [pickerOpen, setPickerOpen] = useState(false)
  const [editing, setEditing] = useState<IndexerResource | null>(null)
  const [addingSchema, setAddingSchema] = useState<ProviderSchema | null>(null)

  const { data: schemas = [] } = useIndexerSchema()
  const { data: indexers = [] } = useIndexersV3()
  const create = useCreateIndexer()
  const update = useUpdateIndexer()
  const del = useDeleteIndexerV3()

  // pickerOpen → picker → addingSchema → settings-modal (add)
  // editing row → settings-modal (edit, schema resolved by implementation)

  // renderCard = IndexerCard(item, onToggle: field => bool)
  // IndexerCard shows name, implementation, enableRss, enableAutomaticSearch,
  //   enableInteractiveSearch as inline toggle pills, priority, edit + delete.

  return (
    <div>
      <h1>Indexers</h1>
      <ProviderListSection
        title="Indexers"
        items={indexers}
        renderCard={(i) => <IndexerCard ... />}
        onAdd={() => setPickerOpen(true)}
        onEdit={setEditing}
        onDelete={(i) => confirm(...) && del.mutate(i.id)}
      />
      {pickerOpen && <ProviderPickerModal
        isOpen providers={schemas}
        onPick={(s) => { setAddingSchema(s); setPickerOpen(false) }}
        onCancel={() => setPickerOpen(false)} />}
      {addingSchema && <ProviderSettingsModal ... />}
      {editing && <ProviderSettingsModal ... />}
    </div>
  )
}
```

### 7.2 `frontend/src/pages/SettingsDownloadClients.tsx`

Same shape as Indexers, plus a `RemotePathMappings` subsection. ~120 lines. Follows the T15 pattern (Settings → Media Management) for the sub-panel styling.

### 7.3 `frontend/src/components/RemotePathMappingsPanel.tsx`

New component bundled into Download Clients page.
- Reads from `useRemotePathMappings()`
- Table: **Host / Remote Path / Local Path / Delete**
- Add row button → inline input row with 3 fields → Save → `useCreateRemotePathMapping()`
- Delete confirms and calls `useDeleteRemotePathMapping()`

Kept dead simple; Sonarr's UI here is just a table and I'm not adding modal ceremony.

## 8. Error handling

- Create/Update failure on POST → toast at page level (use `alert(err.message)` for simplicity, matching T15 pattern). Form stays open so user can retry.
- Delete 409 (not currently returned by the indexer/download-client endpoints — only rootfolder does this) → if added later, follows the `ApiError.details` pattern from the recent apiV3 fix.
- Schema fetch failure → page shows "Failed to load providers" banner with Retry.
- Missing `sonarr2_api_key` → existing `apiFetchRaw` sends no header → backend 401 → `ApiError.message = "Unauthorized"` displayed as toast; user is already logged in so this shouldn't normally fire.

## 9. Testing

No vitest harness yet (confirmed in sub-project #1). Verification for this sub-project is:
- `npm run build` clean
- Manual smoke (during sub-project #2's final task): pick Newznab in the picker, fill in a fake URL + API key, save, confirm it appears in the list; edit it, change the URL, save; delete it.

When vitest lands (future sub-project), add `SchemaFormField.test.tsx`, `ProviderSettingsModal.test.tsx`, `ProviderListSection.test.tsx` covering: render each field type, validation blocks submit on empty required fields, edit seeds values from row.

## 10. Out of scope / follow-ups

- `/test` endpoints per provider (backend) + Test All / per-row Test UI buttons.
- Indexers → Options section (host-config-backed).
- Download Clients → Completed Download Handling toggles (likely host-config).
- Add Category / Tag selectors to field types (right now we treat `select`/`multiselect` as plain text).
- `useIndexers` / `useDownloadClients` (v6) stay alongside the new v3 variants; migrate other call sites in a follow-up once v6 is retired.

## 11. Deliverables

Frontend (no backend):
- [ ] `frontend/src/api/types.ts` — add 5 new types (ProviderFieldSchema, ProviderSchema, IndexerResource, DownloadClientResource, RemotePathMapping)
- [ ] `frontend/src/api/hooks.ts` — add 13 new hooks
- [ ] `frontend/src/components/SchemaFormField.tsx` + `.module.css`
- [ ] `frontend/src/components/ProviderPickerModal.tsx` + `.module.css`
- [ ] `frontend/src/components/ProviderSettingsModal.tsx` + `.module.css`
- [ ] `frontend/src/components/ProviderListSection.tsx` + `.module.css`
- [ ] `frontend/src/components/RemotePathMappingsPanel.tsx` + `.module.css`
- [ ] `frontend/src/pages/SettingsIndexers.tsx` — full rewrite
- [ ] `frontend/src/pages/SettingsDownloadClients.tsx` — full rewrite

Docs:
- [ ] README bullet noting Indexers + Download Clients are now configurable
- [ ] Update `docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md` status
- [ ] Implementation plan at `docs/superpowers/plans/2026-04-14-provider-settings-pages.md`
