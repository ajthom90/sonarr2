import { useState } from 'react'
import {
  useDownloadClientSchema,
  useDownloadClientsV3,
  useCreateDownloadClient,
  useUpdateDownloadClient,
  useDeleteDownloadClientV3,
} from '../api/hooks'
import type {
  DownloadClientResource,
  ProviderSchema,
} from '../api/types'
import { ProviderListSection } from '../components/ProviderListSection'
import { ProviderPickerModal } from '../components/ProviderPickerModal'
import { ProviderSettingsModal } from '../components/ProviderSettingsModal'
import { RemotePathMappingsPanel } from '../components/RemotePathMappingsPanel'
import styles from './SettingsDownloadClients.module.css'

interface DCExtras {
  enable: boolean
  priority: number
}

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

  async function handleCreate(
    name: string,
    fields: Record<string, unknown>,
    extras: DCExtras,
  ) {
    if (!pickedSchema) return
    try {
      await create.mutateAsync({
        name,
        implementation: pickedSchema.implementation,
        fields,
        enable: extras.enable,
        priority: extras.priority,
      })
      setPickedSchema(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleUpdate(
    name: string,
    fields: Record<string, unknown>,
    extras: DCExtras,
  ) {
    if (!editing) return
    try {
      await update.mutateAsync({
        ...editing,
        name,
        fields,
        enable: extras.enable,
        priority: extras.priority,
      })
      setEditing(null)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleDelete(item: DownloadClientResource) {
    if (!confirm(`Remove ${item.name}?`)) return
    try {
      await del.mutateAsync(item.id)
    } catch (err) {
      alert((err as Error).message)
    }
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
            <button
              className={styles.deleteBtn}
              onClick={() => handleDelete(item)}
            >
              Delete
            </button>
          </div>
        </div>
        <div className={styles.cardFlags}>
          <span className={item.enable ? styles.flagOn : styles.flagOff}>
            Enabled
          </span>
          <span className={styles.priority}>Priority: {item.priority}</span>
        </div>
      </div>
    )
  }

  function renderExtras(extras: DCExtras, set: (e: DCExtras) => void) {
    return (
      <>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.enable}
            onChange={(e) => set({ ...extras, enable: e.target.checked })}
          />
          Enable
        </label>
        <label className={styles.extraRow}>
          Priority
          <input
            type="number"
            className={styles.priorityInput}
            value={extras.priority}
            onChange={(e) =>
              set({ ...extras, priority: Number(e.target.value) })
            }
          />
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
          onPick={(s) => {
            setPickerOpen(false)
            setPickedSchema(s)
          }}
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
          onSubmit={({ name, fields, extras }) =>
            handleCreate(name, fields, extras)
          }
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
          onSubmit={({ name, fields, extras }) =>
            handleUpdate(name, fields, extras)
          }
          onCancel={() => setEditing(null)}
        />
      )}
    </div>
  )
}
