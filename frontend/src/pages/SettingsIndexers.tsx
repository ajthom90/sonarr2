import { useState } from 'react'
import {
  useIndexerSchema,
  useIndexersV3,
  useCreateIndexer,
  useUpdateIndexer,
  useDeleteIndexerV3,
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

  async function handleCreate(
    name: string,
    fields: Record<string, unknown>,
    extras: IndexerExtras,
  ) {
    if (!pickedSchema) return
    try {
      await create.mutateAsync({
        name,
        implementation: pickedSchema.implementation,
        fields,
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

  async function handleUpdate(
    name: string,
    fields: Record<string, unknown>,
    extras: IndexerExtras,
  ) {
    if (!editing) return
    try {
      await update.mutateAsync({
        ...editing,
        name,
        fields,
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
    try {
      await del.mutateAsync(item.id)
    } catch (err) {
      alert((err as Error).message)
    }
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
            <button
              className={styles.deleteBtn}
              onClick={() => handleDelete(item)}
            >
              Delete
            </button>
          </div>
        </div>
        <div className={styles.cardFlags}>
          <span className={item.enableRss ? styles.flagOn : styles.flagOff}>
            RSS
          </span>
          <span
            className={
              item.enableAutomaticSearch ? styles.flagOn : styles.flagOff
            }
          >
            Auto Search
          </span>
          <span
            className={
              item.enableInteractiveSearch ? styles.flagOn : styles.flagOff
            }
          >
            Interactive
          </span>
          <span className={styles.priority}>Priority: {item.priority}</span>
        </div>
      </div>
    )
  }

  function renderIndexerExtras(
    extras: IndexerExtras,
    set: (e: IndexerExtras) => void,
  ) {
    return (
      <>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.enableRss}
            onChange={(e) => set({ ...extras, enableRss: e.target.checked })}
          />
          Enable RSS
        </label>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.enableAutomaticSearch}
            onChange={(e) =>
              set({ ...extras, enableAutomaticSearch: e.target.checked })
            }
          />
          Enable Automatic Search
        </label>
        <label className={styles.extraRow}>
          <input
            type="checkbox"
            checked={extras.enableInteractiveSearch}
            onChange={(e) =>
              set({ ...extras, enableInteractiveSearch: e.target.checked })
            }
          />
          Enable Interactive Search
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
          onPick={(s) => {
            setPickerOpen(false)
            setPickedSchema(s)
          }}
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
          extras={{
            enableRss: true,
            enableAutomaticSearch: true,
            enableInteractiveSearch: true,
            priority: 25,
          }}
          renderExtras={renderIndexerExtras}
          onSubmit={({ name, fields, extras }) =>
            handleCreate(name, fields, extras)
          }
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
          onSubmit={({ name, fields, extras }) =>
            handleUpdate(name, fields, extras)
          }
          onCancel={() => setEditing(null)}
        />
      )}
    </div>
  )
}
