import { useState } from 'react'
import {
  useCustomFormats,
  useCreateCustomFormat,
  useUpdateCustomFormat,
  useDeleteCustomFormat,
} from '../api/hooks'
import { Modal } from '../components/Modal'
import type { CustomFormat, CustomFormatSpec } from '../api/types'
import styles from './SettingsCustomFormats.module.css'

// ── Spec editor row ──────────────────────────────────────────────────────────

interface SpecRowProps {
  spec: CustomFormatSpec
  onChange: (updated: CustomFormatSpec) => void
  onRemove: () => void
}

function SpecRow({ spec, onChange, onRemove }: SpecRowProps) {
  const value = spec.fields.find((f) => f.name === 'value')?.value ?? ''

  return (
    <div className={styles.specRow}>
      <input
        className={styles.specInput}
        type="text"
        placeholder="Spec name"
        value={spec.name}
        onChange={(e) => onChange({ ...spec, name: e.target.value })}
      />
      <select
        className={styles.specSelect}
        value={spec.implementation}
        onChange={(e) => onChange({ ...spec, implementation: e.target.value })}
      >
        <option value="ReleaseTitleSpecification">Release Title</option>
        <option value="SourceSpecification">Source</option>
        <option value="ResolutionSpecification">Resolution</option>
        <option value="LanguageSpecification">Language</option>
        <option value="IndexerFlagSpecification">Indexer Flag</option>
      </select>
      <input
        className={styles.specInput}
        type="text"
        placeholder="Regex / value"
        value={value}
        onChange={(e) =>
          onChange({
            ...spec,
            fields: [{ name: 'value', value: e.target.value }],
          })
        }
      />
      <label className={styles.specCheck}>
        <input
          type="checkbox"
          checked={spec.negate}
          onChange={(e) => onChange({ ...spec, negate: e.target.checked })}
        />
        Negate
      </label>
      <label className={styles.specCheck}>
        <input
          type="checkbox"
          checked={spec.required}
          onChange={(e) => onChange({ ...spec, required: e.target.checked })}
        />
        Required
      </label>
      <button className={styles.removeSpecBtn} type="button" onClick={onRemove}>
        &times;
      </button>
    </div>
  )
}

// ── Custom format form modal ─────────────────────────────────────────────────

const emptySpec: CustomFormatSpec = {
  name: '',
  implementation: 'ReleaseTitleSpecification',
  negate: false,
  required: false,
  fields: [{ name: 'value', value: '' }],
}

interface FormModalProps {
  open: boolean
  onClose: () => void
  initial?: CustomFormat
}

function CustomFormatFormModal({ open, onClose, initial }: FormModalProps) {
  const createMutation = useCreateCustomFormat()
  const updateMutation = useUpdateCustomFormat()
  const isEdit = !!initial

  const [name, setName] = useState(initial?.name ?? '')
  const [includeRenaming, setIncludeRenaming] = useState(
    initial?.includeCustomFormatWhenRenaming ?? false,
  )
  const [specs, setSpecs] = useState<CustomFormatSpec[]>(
    initial?.specifications ?? [{ ...emptySpec }],
  )
  const [error, setError] = useState<string | null>(null)

  function resetForm() {
    setName(initial?.name ?? '')
    setIncludeRenaming(initial?.includeCustomFormatWhenRenaming ?? false)
    setSpecs(initial?.specifications ?? [{ ...emptySpec }])
    setError(null)
  }

  function handleClose() {
    resetForm()
    onClose()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    if (!name.trim()) {
      setError('Name is required')
      return
    }

    const body = {
      name: name.trim(),
      includeCustomFormatWhenRenaming: includeRenaming,
      specifications: specs.filter((s) => s.name.trim() !== ''),
    }

    if (isEdit && initial) {
      updateMutation.mutate(
        { id: initial.id, ...body },
        { onSuccess: handleClose, onError: (err) => setError(err instanceof Error ? err.message : 'Failed to save') },
      )
    } else {
      createMutation.mutate(body, {
        onSuccess: handleClose,
        onError: (err) => setError(err instanceof Error ? err.message : 'Failed to create'),
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Modal open={open} onClose={handleClose} title={isEdit ? 'Edit Custom Format' : 'New Custom Format'}>
      <form onSubmit={handleSubmit}>
        <div className={styles.field}>
          <label className={styles.label}>Name</label>
          <input
            className={styles.input}
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Custom format name"
          />
        </div>

        <div className={styles.field}>
          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={includeRenaming}
              onChange={(e) => setIncludeRenaming(e.target.checked)}
              className={styles.checkbox}
            />
            Include custom format when renaming
          </label>
        </div>

        <div className={styles.specsSection}>
          <div className={styles.specsHeader}>
            <span className={styles.specsTitle}>Specifications</span>
            <button
              type="button"
              className={styles.addSpecBtn}
              onClick={() => setSpecs((prev) => [...prev, { ...emptySpec }])}
            >
              + Add
            </button>
          </div>
          {specs.length === 0 && (
            <p className={styles.emptySpecs}>No specifications. Add one above.</p>
          )}
          {specs.map((spec, i) => (
            <SpecRow
              key={i}
              spec={spec}
              onChange={(updated) => {
                const next = [...specs]
                next[i] = updated
                setSpecs(next)
              }}
              onRemove={() => setSpecs((prev) => prev.filter((_, j) => j !== i))}
            />
          ))}
        </div>

        {error && <p className={styles.error}>{error}</p>}

        <div className={styles.actions}>
          <button type="submit" className={styles.saveBtn} disabled={isPending}>
            {isPending ? 'Saving...' : 'Save'}
          </button>
          <button type="button" className={styles.cancelBtn} onClick={handleClose}>
            Cancel
          </button>
        </div>
      </form>
    </Modal>
  )
}

// ── Main page ────────────────────────────────────────────────────────────────

export function SettingsCustomFormats() {
  const { data: customFormats, isLoading, isError, error } = useCustomFormats()
  const deleteMutation = useDeleteCustomFormat()

  const [showForm, setShowForm] = useState(false)
  const [editTarget, setEditTarget] = useState<CustomFormat | undefined>(undefined)

  const formats: CustomFormat[] = customFormats ?? []

  function openCreate() {
    setEditTarget(undefined)
    setShowForm(true)
  }

  function openEdit(cf: CustomFormat) {
    setEditTarget(cf)
    setShowForm(true)
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id)
  }

  if (isLoading) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Custom Formats</h1>
        <p className={styles.stateMessage}>Loading custom formats...</p>
      </div>
    )
  }

  if (isError) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Custom Formats</h1>
        <p className={styles.errorMessage}>
          Failed to load custom formats: {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <h1 className={styles.heading}>Custom Formats</h1>
        <button className={styles.addBtn} onClick={openCreate}>
          + New Custom Format
        </button>
      </div>

      {formats.length === 0 ? (
        <p className={styles.stateMessage}>No custom formats configured.</p>
      ) : (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Name</th>
                <th className={styles.th}>Include in Rename</th>
                <th className={styles.th}>Specifications</th>
                <th className={styles.th}></th>
              </tr>
            </thead>
            <tbody>
              {formats.map((cf) => (
                <tr key={cf.id} className={styles.row}>
                  <td className={styles.td}>
                    <button className={styles.nameBtn} onClick={() => openEdit(cf)}>
                      {cf.name}
                    </button>
                  </td>
                  <td className={styles.td}>
                    <span
                      className={`${styles.pill} ${cf.includeCustomFormatWhenRenaming ? styles.pillEnabled : styles.pillDisabled}`}
                    >
                      {cf.includeCustomFormatWhenRenaming ? 'Yes' : 'No'}
                    </span>
                  </td>
                  <td className={styles.td}>
                    <span className={styles.muted}>{cf.specifications.length}</span>
                  </td>
                  <td className={styles.td}>
                    <div className={styles.rowActions}>
                      <button className={styles.editRowBtn} onClick={() => openEdit(cf)}>
                        Edit
                      </button>
                      <button
                        className={styles.deleteBtn}
                        onClick={() => handleDelete(cf.id)}
                        disabled={deleteMutation.isPending}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <CustomFormatFormModal
        open={showForm}
        onClose={() => setShowForm(false)}
        initial={editTarget}
      />
    </div>
  )
}
