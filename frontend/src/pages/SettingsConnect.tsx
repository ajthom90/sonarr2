import { useState } from 'react'
import {
  useNotificationSchema,
  useNotifications,
  useCreateNotification,
  useUpdateNotification,
  useDeleteNotification,
} from '../api/hooks'
import type { NotificationResource, ProviderSchema } from '../api/types'
import { ProviderListSection } from '../components/ProviderListSection'
import { ProviderPickerModal } from '../components/ProviderPickerModal'
import { ProviderSettingsModal } from '../components/ProviderSettingsModal'
import styles from './SettingsConnect.module.css'

interface NotificationExtras {
  onGrab: boolean
  onDownload: boolean
  onHealthIssue: boolean
  tags: number[]
}

function formatTagsInput(tags: number[]): string {
  return tags.join(', ')
}

function parseTagsInput(raw: string): number[] {
  return raw
    .split(',')
    .map((s) => Number(s.trim()))
    .filter((n) => Number.isFinite(n) && !Number.isNaN(n) && n > 0)
}

export function SettingsConnect() {
  const { data: schemas = [] } = useNotificationSchema()
  const { data: notifications = [] } = useNotifications()
  const create = useCreateNotification()
  const update = useUpdateNotification()
  const del = useDeleteNotification()

  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickedSchema, setPickedSchema] = useState<ProviderSchema | null>(null)
  const [editing, setEditing] = useState<NotificationResource | null>(null)

  const editingSchema = editing
    ? schemas.find((s) => s.implementation === editing.implementation) ?? null
    : null

  async function handleCreate(
    name: string,
    fields: Record<string, unknown>,
    extras: NotificationExtras,
  ) {
    if (!pickedSchema) return
    try {
      await create.mutateAsync({
        name,
        implementation: pickedSchema.implementation,
        fields,
        onGrab: extras.onGrab,
        onDownload: extras.onDownload,
        onHealthIssue: extras.onHealthIssue,
        tags: extras.tags,
      })
      setPickedSchema(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleUpdate(
    name: string,
    fields: Record<string, unknown>,
    extras: NotificationExtras,
  ) {
    if (!editing) return
    try {
      await update.mutateAsync({
        ...editing,
        name,
        fields,
        onGrab: extras.onGrab,
        onDownload: extras.onDownload,
        onHealthIssue: extras.onHealthIssue,
        tags: extras.tags,
      })
      setEditing(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleDelete(item: NotificationResource) {
    if (!confirm(`Remove ${item.name}?`)) return
    try {
      await del.mutateAsync(item.id)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  function renderCard(item: NotificationResource) {
    return (
      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <div>
            <div className={styles.cardName}>{item.name}</div>
            <div className={styles.cardImpl}>{item.implementation}</div>
          </div>
          <div className={styles.cardActions}>
            <button onClick={() => setEditing(item)}>Edit</button>
            <button className={styles.deleteBtn} onClick={() => handleDelete(item)}>
              Delete
            </button>
          </div>
        </div>
        <div className={styles.cardFlags}>
          <span className={item.onGrab ? styles.flagOn : styles.flagOff}>
            On Grab
          </span>
          <span className={item.onDownload ? styles.flagOn : styles.flagOff}>
            On Download
          </span>
          <span className={item.onHealthIssue ? styles.flagOn : styles.flagOff}>
            On Health Issue
          </span>
          {item.tags.length > 0 && (
            <span className={styles.tagsChip}>Tags: {item.tags.join(', ')}</span>
          )}
        </div>
      </div>
    )
  }

  function renderExtras(
    extras: NotificationExtras,
    set: (e: NotificationExtras) => void,
  ) {
    return (
      <>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.onGrab}
            onChange={(e) => set({ ...extras, onGrab: e.target.checked })}
          />
          On Grab
        </label>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.onDownload}
            onChange={(e) => set({ ...extras, onDownload: e.target.checked })}
          />
          On Download
        </label>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.onHealthIssue}
            onChange={(e) => set({ ...extras, onHealthIssue: e.target.checked })}
          />
          On Health Issue
        </label>
        <label className={styles.extraRow}>
          Tags
          <input
            type="text"
            className={styles.tagsInput}
            placeholder="e.g. 1, 2, 5"
            value={formatTagsInput(extras.tags)}
            onChange={(e) => set({ ...extras, tags: parseTagsInput(e.target.value) })}
          />
        </label>
        <p className={styles.extrasHelp}>
          Tags are entered as a comma-separated list of tag IDs. A dedicated tag
          picker is coming in a follow-up.
        </p>
      </>
    )
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Connect</h1>
      <p className={styles.subtitle}>
        Notification integrations fire on grab/download/health events.
        sonarr2 currently supports three event triggers; the full 13-event
        set from Sonarr is a follow-up.
      </p>

      <ProviderListSection<NotificationResource>
        title="Connections"
        items={notifications}
        emptyMessage="No notification integrations yet. Click + Add to wire one up."
        renderCard={renderCard}
        onAdd={() => setPickerOpen(true)}
      />

      {pickerOpen && (
        <ProviderPickerModal
          isOpen
          title="Add Notification"
          providers={schemas}
          onPick={(s) => {
            setPickerOpen(false)
            setPickedSchema(s)
          }}
          onCancel={() => setPickerOpen(false)}
        />
      )}

      {pickedSchema && (
        <ProviderSettingsModal<NotificationExtras>
          isOpen
          title={`Add ${pickedSchema.name} Notification`}
          schema={pickedSchema}
          initialValues={{}}
          initialName={pickedSchema.name}
          extras={{
            onGrab: true,
            onDownload: true,
            onHealthIssue: true,
            tags: [],
          }}
          renderExtras={renderExtras}
          onSubmit={({ name, fields, extras }) =>
            handleCreate(name, fields, extras)
          }
          onCancel={() => setPickedSchema(null)}
        />
      )}

      {editing && editingSchema && (
        <ProviderSettingsModal<NotificationExtras>
          isOpen
          title={`Edit ${editing.name}`}
          schema={editingSchema}
          initialValues={editing.fields}
          initialName={editing.name}
          extras={{
            onGrab: editing.onGrab,
            onDownload: editing.onDownload,
            onHealthIssue: editing.onHealthIssue,
            tags: editing.tags ?? [],
          }}
          renderExtras={renderExtras}
          onSubmit={({ name, fields, extras }) =>
            handleUpdate(name, fields, extras)
          }
          onCancel={() => setEditing(null)}
        />
      )}
    </div>
  )
}
