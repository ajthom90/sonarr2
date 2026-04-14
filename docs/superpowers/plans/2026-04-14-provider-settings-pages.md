# Provider Settings Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Sonarr-parity `/settings/indexers` and `/settings/downloadclients` pages with schema-driven add/edit/delete flows, plus a Remote Path Mappings sub-panel on Download Clients.

**Architecture:** Frontend-only — backend is already wired (10 indexer + 20 download-client providers, `/schema` endpoints, CRUD per v3). Shared components (`SchemaFormField`, `ProviderPickerModal`, `ProviderSettingsModal`, `ProviderListSection`, `RemotePathMappingsPanel`) are generic over provider kind; two thin page wrappers consume them.

**Tech Stack:** React + TypeScript + Vite, React Query, CSS Modules, `apiV3` helper from sub-project #1.

**Spec:** [2026-04-14-provider-settings-pages-design.md](../specs/2026-04-14-provider-settings-pages-design.md)

---

## Task 1: Extend types + hooks

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/hooks.ts`

- [ ] **Step 1: Add the 5 new types to `types.ts`** (append at end):

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

- [ ] **Step 2: Add 13 hooks to `hooks.ts`** (append at end, also extend imports at top to bring in the new types):

```ts
// Indexer schema + v3 CRUD
export function useIndexerSchema() {
  return useQuery({
    queryKey: ['v3', 'indexer', 'schema'],
    queryFn: () => apiV3.get<ProviderSchema[]>('/indexer/schema'),
  })
}

export function useIndexersV3() {
  return useQuery({
    queryKey: ['v3', 'indexer'],
    queryFn: () => apiV3.get<IndexerResource[]>('/indexer'),
  })
}

export function useCreateIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<IndexerResource, 'id' | 'added'>) =>
      apiV3.post<IndexerResource>('/indexer', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

export function useUpdateIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: IndexerResource) =>
      apiV3.post<IndexerResource>(`/indexer/${id}`, body),
    // Note: apiV3 lacks `put`; add it in Step 3 and switch to put here.
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

export function useDeleteIndexerV3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/indexer/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

// Download client schema + v3 CRUD
export function useDownloadClientSchema() {
  return useQuery({
    queryKey: ['v3', 'downloadclient', 'schema'],
    queryFn: () => apiV3.get<ProviderSchema[]>('/downloadclient/schema'),
  })
}

export function useDownloadClientsV3() {
  return useQuery({
    queryKey: ['v3', 'downloadclient'],
    queryFn: () => apiV3.get<DownloadClientResource[]>('/downloadclient'),
  })
}

export function useCreateDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<DownloadClientResource, 'id' | 'added'>) =>
      apiV3.post<DownloadClientResource>('/downloadclient', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

export function useUpdateDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: DownloadClientResource) =>
      apiV3.put<DownloadClientResource>(`/downloadclient/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

export function useDeleteDownloadClientV3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/downloadclient/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

// Remote path mappings
export function useRemotePathMappings() {
  return useQuery({
    queryKey: ['v3', 'remotepathmapping'],
    queryFn: () => apiV3.get<RemotePathMapping[]>('/remotepathmapping'),
  })
}

export function useCreateRemotePathMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<RemotePathMapping, 'id'>) =>
      apiV3.post<RemotePathMapping>('/remotepathmapping', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'remotepathmapping'] }),
  })
}

export function useDeleteRemotePathMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/remotepathmapping/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'remotepathmapping'] }),
  })
}
```

- [ ] **Step 3: Add `put` to `apiV3` helper**

`apiV3` from T12 currently exposes `get`/`post`/`delete` — no `put`. Extend `frontend/src/api/v3.ts`:

```ts
export const apiV3 = {
  get: <T>(path: string) => v3<T>(path),
  post: <T>(path: string, body?: unknown) =>
    v3<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body: unknown) =>
    v3<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (path: string) => v3<void>(path, { method: 'DELETE' }),
}
```

Then update `useUpdateIndexer` to use `apiV3.put` instead of the `apiV3.post` placeholder.

- [ ] **Step 4: Verify build**

```bash
cd frontend && npm run build
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/hooks.ts frontend/src/api/v3.ts
git commit -m "feat(frontend): add provider schema + indexer/downloadclient/remotepathmapping v3 hooks"
```

---

## Task 2: `SchemaFormField` component

**Files:**
- Create: `frontend/src/components/SchemaFormField.tsx`
- Create: `frontend/src/components/SchemaFormField.module.css`

- [ ] **Step 1: Implement**

`SchemaFormField.tsx`:
```tsx
import type { ProviderFieldSchema } from '../api/types'
import styles from './SchemaFormField.module.css'

export interface SchemaFormFieldProps {
  schema: ProviderFieldSchema
  value: unknown
  onChange: (next: unknown) => void
}

export function SchemaFormField({ schema, value, onChange }: SchemaFormFieldProps) {
  const id = `field-${schema.name}`

  function renderInput() {
    switch (schema.type) {
      case 'text':
        return (
          <input id={id} type="text" className={styles.input}
            placeholder={schema.placeholder}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)} />
        )
      case 'password':
        return (
          <input id={id} type="password" className={styles.input}
            placeholder={schema.placeholder}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)} />
        )
      case 'number':
        return (
          <input id={id} type="number" className={styles.input}
            placeholder={schema.placeholder}
            value={(value as number) ?? ''}
            onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))} />
        )
      case 'checkbox':
        return (
          <input id={id} type="checkbox" className={styles.checkbox}
            checked={Boolean(value)}
            onChange={(e) => onChange(e.target.checked)} />
        )
      case 'select':
      case 'multiselect':
        // Backend FieldSchema doesn't emit options yet. Fall back to text.
        // TODO: when backend adds options[], render real <select>.
        return (
          <input id={id} type="text" className={styles.input}
            placeholder={schema.placeholder ?? '(enter value)'}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)} />
        )
      default:
        return <span>Unsupported field type: {schema.type}</span>
    }
  }

  return (
    <div className={styles.field}>
      <label htmlFor={id} className={styles.label}>
        {schema.label || schema.name}
        {schema.required && <span className={styles.required}> *</span>}
      </label>
      {renderInput()}
      {schema.helpText && <p className={styles.help}>{schema.helpText}</p>}
    </div>
  )
}
```

`SchemaFormField.module.css`:
```css
.field { display: flex; flex-direction: column; gap: 4px; margin-bottom: 12px; }
.label { font-size: 0.85rem; color: var(--color-text-muted, #9aa0a6); }
.required { color: var(--color-danger, #ef4444); }
.input {
  padding: 6px 10px;
  border-radius: 4px;
  border: 1px solid var(--color-border, #2d3138);
  background: var(--color-bg-surface, #1b1e22);
  color: inherit;
  font-size: 0.95rem;
}
.checkbox { width: 18px; height: 18px; }
.help { font-size: 0.8rem; color: var(--color-text-muted, #9aa0a6); margin: 2px 0 0; }
```

- [ ] **Step 2: Verify build + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/components/SchemaFormField.tsx frontend/src/components/SchemaFormField.module.css
git commit -m "feat(frontend): add SchemaFormField component"
```

---

## Task 3: `ProviderPickerModal` component

**Files:**
- Create: `frontend/src/components/ProviderPickerModal.tsx`
- Create: `frontend/src/components/ProviderPickerModal.module.css`

- [ ] **Step 1: Implement**

```tsx
import { useEffect } from 'react'
import type { ProviderSchema } from '../api/types'
import styles from './ProviderPickerModal.module.css'

export interface ProviderPickerModalProps {
  isOpen: boolean
  title: string
  providers: ProviderSchema[]
  onPick: (schema: ProviderSchema) => void
  onCancel: () => void
}

export function ProviderPickerModal({
  isOpen, title, providers, onPick, onCancel,
}: ProviderPickerModalProps) {
  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') onCancel() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, onCancel])

  if (!isOpen) return null

  return (
    <div className={styles.backdrop} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.header}>
          <h2>{title}</h2>
          <button aria-label="Close" className={styles.close} onClick={onCancel}>×</button>
        </header>
        <ul className={styles.list}>
          {providers.map((p) => (
            <li key={p.implementation}>
              <button type="button" className={styles.row} onClick={() => onPick(p)}>
                <div className={styles.name}>{p.name}</div>
                <div className={styles.impl}>{p.implementation}</div>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
```

CSS — mirror the backdrop/modal/header/close pattern from T13's `FileBrowserModal.module.css`, add:
```css
.list { list-style: none; margin: 0; padding: 8px; overflow-y: auto; max-height: 60vh;
  display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 8px; }
.row { display: block; width: 100%; padding: 12px; background: var(--color-surface-hover, #242830);
  border: 1px solid var(--color-border, #2d3138); border-radius: 4px; color: inherit;
  cursor: pointer; text-align: left; }
.row:hover { border-color: var(--color-accent, #3b82f6); }
.name { font-weight: 600; margin-bottom: 4px; }
.impl { font-size: 0.8rem; color: var(--color-text-muted, #9aa0a6); }
```

(Copy backdrop/modal/header/close from FileBrowserModal.module.css.)

- [ ] **Step 2: Verify + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/components/ProviderPickerModal.tsx frontend/src/components/ProviderPickerModal.module.css
git commit -m "feat(frontend): add ProviderPickerModal component"
```

---

## Task 4: `ProviderSettingsModal` component

**Files:**
- Create: `frontend/src/components/ProviderSettingsModal.tsx`
- Create: `frontend/src/components/ProviderSettingsModal.module.css`

- [ ] **Step 1: Implement**

```tsx
import { useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { ProviderSchema } from '../api/types'
import { SchemaFormField } from './SchemaFormField'
import styles from './ProviderSettingsModal.module.css'

export interface ProviderSettingsPayload<Extras> {
  name: string
  fields: Record<string, unknown>
  extras: Extras
}

export interface ProviderSettingsModalProps<Extras> {
  isOpen: boolean
  title: string
  schema: ProviderSchema
  initialValues: Record<string, unknown>
  initialName: string
  extras: Extras
  renderExtras: (extras: Extras, set: (e: Extras) => void) => ReactNode
  onSubmit: (payload: ProviderSettingsPayload<Extras>) => void
  onCancel: () => void
}

export function ProviderSettingsModal<Extras>({
  isOpen, title, schema, initialValues, initialName, extras, renderExtras, onSubmit, onCancel,
}: ProviderSettingsModalProps<Extras>) {
  const [name, setName] = useState(initialName)
  const [values, setValues] = useState<Record<string, unknown>>(() => {
    const next: Record<string, unknown> = {}
    for (const f of schema.fields) {
      if (initialValues[f.name] !== undefined) {
        next[f.name] = initialValues[f.name]
      } else if (f.default !== undefined && f.default !== '') {
        // Coerce default string into the field's type.
        next[f.name] = f.type === 'number' ? Number(f.default)
          : f.type === 'checkbox' ? f.default === 'true'
          : f.default
      }
    }
    return next
  })
  const [extrasState, setExtrasState] = useState(extras)
  const [showAdvanced, setShowAdvanced] = useState(false)

  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') onCancel() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, onCancel])

  const basicFields = useMemo(() => schema.fields.filter((f) => !f.advanced), [schema.fields])
  const advancedFields = useMemo(() => schema.fields.filter((f) => f.advanced), [schema.fields])

  const requiredMissing = schema.fields
    .filter((f) => f.required)
    .some((f) => {
      const v = values[f.name]
      return v === undefined || v === null || v === ''
    })
  const canSubmit = name.trim() !== '' && !requiredMissing

  if (!isOpen) return null

  function handleSubmit() {
    onSubmit({ name: name.trim(), fields: values, extras: extrasState })
  }

  return (
    <div className={styles.backdrop} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.header}>
          <h2>{title}</h2>
          <button aria-label="Close" className={styles.close} onClick={onCancel}>×</button>
        </header>

        <div className={styles.body}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="provider-name">Name</label>
            <input id="provider-name" type="text" className={styles.input}
              value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          {basicFields.map((f) => (
            <SchemaFormField key={f.name} schema={f}
              value={values[f.name]}
              onChange={(v) => setValues((s) => ({ ...s, [f.name]: v }))} />
          ))}

          {advancedFields.length > 0 && (
            <>
              <button type="button" className={styles.advancedToggle}
                onClick={() => setShowAdvanced((s) => !s)}>
                {showAdvanced ? 'Hide' : 'Show'} Advanced
              </button>
              {showAdvanced && advancedFields.map((f) => (
                <SchemaFormField key={f.name} schema={f}
                  value={values[f.name]}
                  onChange={(v) => setValues((s) => ({ ...s, [f.name]: v }))} />
              ))}
            </>
          )}

          <div className={styles.extrasSection}>
            {renderExtras(extrasState, setExtrasState)}
          </div>
        </div>

        <footer className={styles.footer}>
          <button type="button" onClick={onCancel}>Cancel</button>
          <button type="button" className={styles.primary}
            disabled={!canSubmit} onClick={handleSubmit}>
            Save
          </button>
        </footer>
      </div>
    </div>
  )
}
```

CSS: copy backdrop/modal/header/close/footer/primary from `FileBrowserModal.module.css` pattern. Add:
```css
.body { padding: 16px; overflow-y: auto; flex: 1; }
.field { display: flex; flex-direction: column; gap: 4px; margin-bottom: 12px; }
.label { font-size: 0.85rem; color: var(--color-text-muted, #9aa0a6); }
.input { padding: 6px 10px; border-radius: 4px; border: 1px solid var(--color-border, #2d3138);
  background: var(--color-bg-surface, #1b1e22); color: inherit; font-size: 0.95rem; }
.advancedToggle { background: none; border: 1px dashed var(--color-border, #2d3138);
  color: var(--color-text-muted, #9aa0a6); padding: 6px 12px; border-radius: 4px;
  cursor: pointer; margin: 8px 0; font-size: 0.9rem; }
.extrasSection { margin-top: 16px; padding-top: 16px;
  border-top: 1px solid var(--color-border, #2d3138); }
```

- [ ] **Step 2: Verify + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/components/ProviderSettingsModal.tsx frontend/src/components/ProviderSettingsModal.module.css
git commit -m "feat(frontend): add ProviderSettingsModal component"
```

---

## Task 5: `ProviderListSection` component

**Files:**
- Create: `frontend/src/components/ProviderListSection.tsx`
- Create: `frontend/src/components/ProviderListSection.module.css`

- [ ] **Step 1: Implement**

```tsx
import type { ReactNode } from 'react'
import styles from './ProviderListSection.module.css'

export interface ProviderListSectionProps<T> {
  title: string
  items: T[]
  emptyMessage: string
  renderCard: (item: T) => ReactNode
  onAdd: () => void
}

export function ProviderListSection<T>({
  title, items, emptyMessage, renderCard, onAdd,
}: ProviderListSectionProps<T>) {
  return (
    <section className={styles.section}>
      <div className={styles.header}>
        <h2 className={styles.title}>{title}</h2>
        <button type="button" className={styles.addBtn} onClick={onAdd}>+ Add</button>
      </div>
      {items.length === 0 ? (
        <p className={styles.empty}>{emptyMessage}</p>
      ) : (
        <div className={styles.grid}>
          {items.map((item, i) => (
            <div key={(item as { id?: number }).id ?? i} className={styles.cardWrap}>
              {renderCard(item)}
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
```

CSS:
```css
.section { margin-bottom: 32px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.title { margin: 0; font-size: 1.1rem; }
.addBtn { padding: 6px 14px; border-radius: 4px; border: 1px solid var(--color-border, #2d3138);
  background: var(--color-accent, #3b82f6); color: white; cursor: pointer; font-size: 0.9rem; }
.empty { color: var(--color-text-muted, #9aa0a6); padding: 24px; text-align: center;
  border: 1px dashed var(--color-border, #2d3138); border-radius: 6px; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.cardWrap { /* cards style themselves */ }
```

- [ ] **Step 2: Verify + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/components/ProviderListSection.tsx frontend/src/components/ProviderListSection.module.css
git commit -m "feat(frontend): add ProviderListSection generic component"
```

---

## Task 6: `SettingsIndexers` page

**Files:**
- Modify (full rewrite): `frontend/src/pages/SettingsIndexers.tsx`
- Create: `frontend/src/pages/SettingsIndexers.module.css`

- [ ] **Step 1: Implement**

Full rewrite of `frontend/src/pages/SettingsIndexers.tsx`:

```tsx
import { useState } from 'react'
import {
  useIndexerSchema, useIndexersV3,
  useCreateIndexer, useUpdateIndexer, useDeleteIndexerV3,
} from '../api/hooks'
import type { IndexerResource, ProviderSchema } from '../api/types'
import { ProviderListSection } from '../components/ProviderListSection'
import { ProviderPickerModal } from '../components/ProviderPickerModal'
import { ProviderSettingsModal } from '../components/ProviderSettingsModal'
import styles from './SettingsIndexers.module.css'

interface IndexerExtras {
  enableRss: boolean
  enableAutomaticSearch: boolean
  enableInteractiveSearch: boolean
  priority: number
}

export function SettingsIndexers() {
  const { data: schemas = [] } = useIndexerSchema()
  const { data: indexers = [] } = useIndexersV3()
  const create = useCreateIndexer()
  const update = useUpdateIndexer()
  const del = useDeleteIndexerV3()

  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickedSchema, setPickedSchema] = useState<ProviderSchema | null>(null)
  const [editing, setEditing] = useState<IndexerResource | null>(null)

  const editingSchema = editing
    ? schemas.find((s) => s.implementation === editing.implementation) ?? null
    : null

  async function handleCreate(name: string, fields: Record<string, unknown>, extras: IndexerExtras) {
    try {
      await create.mutateAsync({
        name, implementation: pickedSchema!.implementation, fields,
        enableRss: extras.enableRss,
        enableAutomaticSearch: extras.enableAutomaticSearch,
        enableInteractiveSearch: extras.enableInteractiveSearch,
        priority: extras.priority,
      })
      setPickedSchema(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleUpdate(name: string, fields: Record<string, unknown>, extras: IndexerExtras) {
    if (!editing) return
    try {
      await update.mutateAsync({
        ...editing, name, fields,
        enableRss: extras.enableRss,
        enableAutomaticSearch: extras.enableAutomaticSearch,
        enableInteractiveSearch: extras.enableInteractiveSearch,
        priority: extras.priority,
      })
      setEditing(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleDelete(item: IndexerResource) {
    if (!confirm(`Remove ${item.name}?`)) return
    try { await del.mutateAsync(item.id) }
    catch (err) { alert((err as Error).message) }
  }

  function renderCard(item: IndexerResource) {
    return (
      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <div>
            <div className={styles.cardName}>{item.name}</div>
            <div className={styles.cardImpl}>{item.implementation}</div>
          </div>
          <div className={styles.cardActions}>
            <button onClick={() => setEditing(item)}>Edit</button>
            <button className={styles.deleteBtn} onClick={() => handleDelete(item)}>Delete</button>
          </div>
        </div>
        <div className={styles.cardFlags}>
          <span className={item.enableRss ? styles.flagOn : styles.flagOff}>RSS</span>
          <span className={item.enableAutomaticSearch ? styles.flagOn : styles.flagOff}>Auto Search</span>
          <span className={item.enableInteractiveSearch ? styles.flagOn : styles.flagOff}>Interactive</span>
          <span className={styles.priority}>Priority: {item.priority}</span>
        </div>
      </div>
    )
  }

  function renderIndexerExtras(extras: IndexerExtras, set: (e: IndexerExtras) => void) {
    return (
      <>
        <label className={styles.extraRow}>
          <input type="checkbox" checked={extras.enableRss}
            onChange={(e) => set({ ...extras, enableRss: e.target.checked })} />
          Enable RSS
        </label>
        <label className={styles.extraRow}>
          <input type="checkbox" checked={extras.enableAutomaticSearch}
            onChange={(e) => set({ ...extras, enableAutomaticSearch: e.target.checked })} />
          Enable Automatic Search
        </label>
        <label className={styles.extraRow}>
          <input type="checkbox" checked={extras.enableInteractiveSearch}
            onChange={(e) => set({ ...extras, enableInteractiveSearch: e.target.checked })} />
          Enable Interactive Search
        </label>
        <label className={styles.extraRow}>
          Priority
          <input type="number" className={styles.priorityInput}
            value={extras.priority}
            onChange={(e) => set({ ...extras, priority: Number(e.target.value) })} />
        </label>
      </>
    )
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Indexers</h1>

      <ProviderListSection<IndexerResource>
        title="Indexers"
        items={indexers}
        emptyMessage="No indexers configured. Click + Add to get started."
        renderCard={renderCard}
        onAdd={() => setPickerOpen(true)}
      />

      {pickerOpen && (
        <ProviderPickerModal
          isOpen
          title="Add Indexer"
          providers={schemas}
          onPick={(s) => { setPickerOpen(false); setPickedSchema(s) }}
          onCancel={() => setPickerOpen(false)}
        />
      )}

      {pickedSchema && (
        <ProviderSettingsModal<IndexerExtras>
          isOpen
          title={`Add ${pickedSchema.name} Indexer`}
          schema={pickedSchema}
          initialValues={{}}
          initialName={pickedSchema.name}
          extras={{ enableRss: true, enableAutomaticSearch: true, enableInteractiveSearch: true, priority: 25 }}
          renderExtras={renderIndexerExtras}
          onSubmit={({ name, fields, extras }) => handleCreate(name, fields, extras)}
          onCancel={() => setPickedSchema(null)}
        />
      )}

      {editing && editingSchema && (
        <ProviderSettingsModal<IndexerExtras>
          isOpen
          title={`Edit ${editing.name}`}
          schema={editingSchema}
          initialValues={editing.fields}
          initialName={editing.name}
          extras={{
            enableRss: editing.enableRss,
            enableAutomaticSearch: editing.enableAutomaticSearch,
            enableInteractiveSearch: editing.enableInteractiveSearch,
            priority: editing.priority,
          }}
          renderExtras={renderIndexerExtras}
          onSubmit={({ name, fields, extras }) => handleUpdate(name, fields, extras)}
          onCancel={() => setEditing(null)}
        />
      )}
    </div>
  )
}
```

CSS:
```css
.page { padding: 24px; max-width: 1200px; margin: 0 auto; }
.pageTitle { font-size: 1.5rem; margin: 0 0 24px; }
.card { background: var(--color-bg-surface, #1b1e22); border: 1px solid var(--color-border, #2d3138);
  border-radius: 6px; padding: 12px 14px; }
.cardHeader { display: flex; justify-content: space-between; gap: 12px; margin-bottom: 8px; }
.cardName { font-weight: 600; }
.cardImpl { font-size: 0.8rem; color: var(--color-text-muted, #9aa0a6); }
.cardActions { display: flex; gap: 6px; }
.cardActions button { padding: 2px 10px; border-radius: 3px; border: 1px solid var(--color-border, #2d3138);
  background: transparent; color: inherit; cursor: pointer; font-size: 0.8rem; }
.deleteBtn { color: var(--color-danger, #ef4444); }
.cardFlags { display: flex; gap: 6px; flex-wrap: wrap; font-size: 0.8rem; }
.flagOn { background: rgba(34, 197, 94, 0.2); color: #22c55e; padding: 2px 6px; border-radius: 3px; }
.flagOff { background: rgba(156, 163, 175, 0.15); color: var(--color-text-muted, #9aa0a6); padding: 2px 6px; border-radius: 3px; }
.priority { color: var(--color-text-muted, #9aa0a6); padding: 2px 6px; }
.extraRow { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; font-size: 0.9rem; }
.priorityInput { width: 80px; padding: 4px 8px; border-radius: 4px;
  border: 1px solid var(--color-border, #2d3138);
  background: var(--color-bg-surface, #1b1e22); color: inherit; }
```

- [ ] **Step 2: Verify + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/pages/SettingsIndexers.tsx frontend/src/pages/SettingsIndexers.module.css
git commit -m "feat(frontend): rewrite /settings/indexers with schema-driven add/edit/delete"
```

---

## Task 7: `SettingsDownloadClients` page + `RemotePathMappingsPanel`

**Files:**
- Modify (full rewrite): `frontend/src/pages/SettingsDownloadClients.tsx`
- Create: `frontend/src/pages/SettingsDownloadClients.module.css`
- Create: `frontend/src/components/RemotePathMappingsPanel.tsx`
- Create: `frontend/src/components/RemotePathMappingsPanel.module.css`

- [ ] **Step 1: Implement `RemotePathMappingsPanel.tsx`**

```tsx
import { useState } from 'react'
import {
  useRemotePathMappings, useCreateRemotePathMapping, useDeleteRemotePathMapping,
} from '../api/hooks'
import styles from './RemotePathMappingsPanel.module.css'

export function RemotePathMappingsPanel() {
  const { data: mappings = [] } = useRemotePathMappings()
  const create = useCreateRemotePathMapping()
  const del = useDeleteRemotePathMapping()

  const [host, setHost] = useState('')
  const [remotePath, setRemotePath] = useState('')
  const [localPath, setLocalPath] = useState('')

  async function handleAdd() {
    if (!host || !remotePath || !localPath) return
    try {
      await create.mutateAsync({ host, remotePath, localPath })
      setHost(''); setRemotePath(''); setLocalPath('')
    } catch (err) { alert((err as Error).message) }
  }

  async function handleDelete(id: number) {
    if (!confirm('Remove this mapping?')) return
    try { await del.mutateAsync(id) }
    catch (err) { alert((err as Error).message) }
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.title}>Remote Path Mappings</h2>
      <p className={styles.help}>
        Translate paths from a download client's host to paths visible on this server.
      </p>

      {mappings.length > 0 ? (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Host</th>
              <th>Remote Path</th>
              <th>Local Path</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {mappings.map((m) => (
              <tr key={m.id}>
                <td>{m.host}</td>
                <td><code>{m.remotePath}</code></td>
                <td><code>{m.localPath}</code></td>
                <td>
                  <button className={styles.deleteBtn} onClick={() => handleDelete(m.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className={styles.empty}>No mappings yet.</p>
      )}

      <div className={styles.addRow}>
        <input placeholder="download-client.host" value={host}
          onChange={(e) => setHost(e.target.value)} />
        <input placeholder="/remote/path" value={remotePath}
          onChange={(e) => setRemotePath(e.target.value)} />
        <input placeholder="/local/path" value={localPath}
          onChange={(e) => setLocalPath(e.target.value)} />
        <button className={styles.addBtn} onClick={handleAdd}>Add</button>
      </div>
    </section>
  )
}
```

CSS:
```css
.section { margin-top: 32px; padding-top: 24px; border-top: 1px solid var(--color-border, #2d3138); }
.title { margin: 0 0 6px; font-size: 1.1rem; }
.help { color: var(--color-text-muted, #9aa0a6); font-size: 0.9rem; margin: 0 0 12px; }
.empty { color: var(--color-text-muted, #9aa0a6); padding: 12px; border: 1px dashed var(--color-border, #2d3138); border-radius: 6px; }
.table { width: 100%; border-collapse: collapse; font-size: 0.9rem; margin-bottom: 12px; }
.table th, .table td { padding: 6px 10px; border-bottom: 1px solid var(--color-border, #2d3138); text-align: left; }
.deleteBtn { padding: 2px 10px; border-radius: 3px; border: 1px solid var(--color-border, #2d3138);
  background: transparent; color: var(--color-danger, #ef4444); cursor: pointer; font-size: 0.8rem; }
.addRow { display: flex; gap: 8px; align-items: center; }
.addRow input { flex: 1; padding: 6px 10px; border-radius: 4px; border: 1px solid var(--color-border, #2d3138);
  background: var(--color-bg-surface, #1b1e22); color: inherit; font-size: 0.9rem; }
.addBtn { padding: 6px 14px; border-radius: 4px; border: none;
  background: var(--color-accent, #3b82f6); color: white; cursor: pointer; font-size: 0.9rem; }
```

- [ ] **Step 2: Implement `SettingsDownloadClients.tsx`**

Mirror the Indexers page structure but:
- Use `useDownloadClientSchema`, `useDownloadClientsV3`, `useCreateDownloadClient`, `useUpdateDownloadClient`, `useDeleteDownloadClientV3`
- Extras type: `{ enable: boolean; priority: number }`
- Card flags: single `Enable` pill + priority
- After the `ProviderListSection`, render `<RemotePathMappingsPanel />`

```tsx
import { useState } from 'react'
import {
  useDownloadClientSchema, useDownloadClientsV3,
  useCreateDownloadClient, useUpdateDownloadClient, useDeleteDownloadClientV3,
} from '../api/hooks'
import type { DownloadClientResource, ProviderSchema } from '../api/types'
import { ProviderListSection } from '../components/ProviderListSection'
import { ProviderPickerModal } from '../components/ProviderPickerModal'
import { ProviderSettingsModal } from '../components/ProviderSettingsModal'
import { RemotePathMappingsPanel } from '../components/RemotePathMappingsPanel'
import styles from './SettingsDownloadClients.module.css'

interface DCExtras { enable: boolean; priority: number }

export function SettingsDownloadClients() {
  const { data: schemas = [] } = useDownloadClientSchema()
  const { data: clients = [] } = useDownloadClientsV3()
  const create = useCreateDownloadClient()
  const update = useUpdateDownloadClient()
  const del = useDeleteDownloadClientV3()

  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickedSchema, setPickedSchema] = useState<ProviderSchema | null>(null)
  const [editing, setEditing] = useState<DownloadClientResource | null>(null)

  const editingSchema = editing
    ? schemas.find((s) => s.implementation === editing.implementation) ?? null
    : null

  async function handleCreate(name: string, fields: Record<string, unknown>, extras: DCExtras) {
    try {
      await create.mutateAsync({
        name, implementation: pickedSchema!.implementation, fields,
        enable: extras.enable, priority: extras.priority,
      })
      setPickedSchema(null)
    } catch (err) { alert((err as Error).message) }
  }

  async function handleUpdate(name: string, fields: Record<string, unknown>, extras: DCExtras) {
    if (!editing) return
    try {
      await update.mutateAsync({
        ...editing, name, fields, enable: extras.enable, priority: extras.priority,
      })
      setEditing(null)
    } catch (err) { alert((err as Error).message) }
  }

  async function handleDelete(item: DownloadClientResource) {
    if (!confirm(`Remove ${item.name}?`)) return
    try { await del.mutateAsync(item.id) }
    catch (err) { alert((err as Error).message) }
  }

  function renderCard(item: DownloadClientResource) {
    return (
      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <div>
            <div className={styles.cardName}>{item.name}</div>
            <div className={styles.cardImpl}>{item.implementation}</div>
          </div>
          <div className={styles.cardActions}>
            <button onClick={() => setEditing(item)}>Edit</button>
            <button className={styles.deleteBtn} onClick={() => handleDelete(item)}>Delete</button>
          </div>
        </div>
        <div className={styles.cardFlags}>
          <span className={item.enable ? styles.flagOn : styles.flagOff}>Enabled</span>
          <span className={styles.priority}>Priority: {item.priority}</span>
        </div>
      </div>
    )
  }

  function renderExtras(extras: DCExtras, set: (e: DCExtras) => void) {
    return (
      <>
        <label className={styles.extraRow}>
          <input type="checkbox" checked={extras.enable}
            onChange={(e) => set({ ...extras, enable: e.target.checked })} />
          Enable
        </label>
        <label className={styles.extraRow}>
          Priority
          <input type="number" className={styles.priorityInput}
            value={extras.priority}
            onChange={(e) => set({ ...extras, priority: Number(e.target.value) })} />
        </label>
      </>
    )
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Download Clients</h1>

      <ProviderListSection<DownloadClientResource>
        title="Download Clients"
        items={clients}
        emptyMessage="No download clients configured. Click + Add to get started."
        renderCard={renderCard}
        onAdd={() => setPickerOpen(true)}
      />

      <RemotePathMappingsPanel />

      {pickerOpen && (
        <ProviderPickerModal
          isOpen
          title="Add Download Client"
          providers={schemas}
          onPick={(s) => { setPickerOpen(false); setPickedSchema(s) }}
          onCancel={() => setPickerOpen(false)}
        />
      )}

      {pickedSchema && (
        <ProviderSettingsModal<DCExtras>
          isOpen
          title={`Add ${pickedSchema.name} Download Client`}
          schema={pickedSchema}
          initialValues={{}}
          initialName={pickedSchema.name}
          extras={{ enable: true, priority: 1 }}
          renderExtras={renderExtras}
          onSubmit={({ name, fields, extras }) => handleCreate(name, fields, extras)}
          onCancel={() => setPickedSchema(null)}
        />
      )}

      {editing && editingSchema && (
        <ProviderSettingsModal<DCExtras>
          isOpen
          title={`Edit ${editing.name}`}
          schema={editingSchema}
          initialValues={editing.fields}
          initialName={editing.name}
          extras={{ enable: editing.enable, priority: editing.priority }}
          renderExtras={renderExtras}
          onSubmit={({ name, fields, extras }) => handleUpdate(name, fields, extras)}
          onCancel={() => setEditing(null)}
        />
      )}
    </div>
  )
}
```

CSS: copy the same pattern as `SettingsIndexers.module.css` (just one `.flagOn` pill for Enable + priority pill).

- [ ] **Step 3: Verify + commit**

```bash
cd frontend && npm run build
cd ..
git add frontend/src/pages/SettingsDownloadClients.tsx frontend/src/pages/SettingsDownloadClients.module.css \
        frontend/src/components/RemotePathMappingsPanel.tsx frontend/src/components/RemotePathMappingsPanel.module.css
git commit -m "feat(frontend): rewrite /settings/downloadclients + add RemotePathMappingsPanel"
```

---

## Task 8: Docs + smoke + push

**Files:**
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md`

- [ ] **Step 1: README bullet**

Add under `What's Implemented`:
```markdown
- **Indexers + Download Clients settings pages** — schema-driven add/edit/delete for all 10 indexer and 20 download-client providers. Download Clients page also has a Remote Path Mappings sub-panel. Test / Test All actions remain deferred pending per-provider test endpoints on the backend.
```

- [ ] **Step 2: M24 doc status table**

Find the row(s) for Indexers, Download Clients, Remote Path Mappings in §2.3 or equivalent. Mark Indexers + Download Clients settings UIs as DONE. Remote Path Mappings UI → DONE (backend was already listed as shipped).

- [ ] **Step 3: Commit docs + push branch**

```bash
git add README.md docs/superpowers/specs/2026-04-15-m24-sonarr-parity-design.md
git commit -m "docs: mark provider settings pages as shipped"
git push origin feature/root-folders-library-import
gh run list --branch feature/root-folders-library-import --limit 3
```

- [ ] **Step 4: Rebuild frontend assets into `web/dist/`**

The existing PR #2 expects `web/dist/index.html` to reference current asset hashes. Rebuild and commit if the hashes drifted:
```bash
cd frontend && npm run build && cd ..
git diff --quiet web/dist/ || (git add web/dist/ && git commit -m "chore: rebuild frontend assets")
git push origin feature/root-folders-library-import
```

- [ ] **Step 5: Watch CI**

```bash
gh run watch --exit-status $(gh run list --branch feature/root-folders-library-import --limit 1 --json databaseId --jq '.[0].databaseId')
```

Expected: green.

---

## Self-review

**Spec coverage:**

| Spec section | Task(s) |
|---|---|
| §4 Types | T1 step 1 |
| §5 Hooks | T1 step 2–3 |
| §6.1 SchemaFormField | T2 |
| §6.2 ProviderSettingsModal | T4 |
| §6.3 ProviderPickerModal | T3 |
| §6.4 ProviderListSection | T5 |
| §7.1 SettingsIndexers | T6 |
| §7.2 SettingsDownloadClients | T7 |
| §7.3 RemotePathMappingsPanel | T7 |
| §8 Error handling | inline in T6/T7 via `alert(err.message)` |
| §9 Testing | none (no vitest harness yet); manual smoke in T8 |
| §10 Follow-ups | not implemented, that's the point |

**Placeholder scan:** no TBDs, no "similar to Task N", every step shows actual code.

**Type consistency:**
- `ProviderSchema.fields` typed as `ProviderFieldSchema[]` everywhere
- `IndexerResource`/`DownloadClientResource` used in both hooks and pages
- `ProviderSettingsPayload<Extras>` generic parameter threads through Task 4 and both page wrappers

No issues found.
