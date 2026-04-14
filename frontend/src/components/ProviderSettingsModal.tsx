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
  isOpen,
  title,
  schema,
  initialValues,
  initialName,
  extras,
  renderExtras,
  onSubmit,
  onCancel,
}: ProviderSettingsModalProps<Extras>) {
  const [name, setName] = useState(initialName)
  const [values, setValues] = useState<Record<string, unknown>>(() => {
    const next: Record<string, unknown> = {}
    for (const f of schema.fields) {
      if (initialValues[f.name] !== undefined) {
        next[f.name] = initialValues[f.name]
      } else if (f.default !== undefined && f.default !== '') {
        // Coerce default string into the field's type.
        next[f.name] =
          f.type === 'number'
            ? Number(f.default)
            : f.type === 'checkbox'
              ? f.default === 'true'
              : f.default
      }
    }
    return next
  })
  const [extrasState, setExtrasState] = useState(extras)
  const [showAdvanced, setShowAdvanced] = useState(false)

  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, onCancel])

  const basicFields = useMemo(
    () => schema.fields.filter((f) => !f.advanced),
    [schema.fields],
  )
  const advancedFields = useMemo(
    () => schema.fields.filter((f) => f.advanced),
    [schema.fields],
  )

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
          <button aria-label="Close" className={styles.close} onClick={onCancel}>
            ×
          </button>
        </header>

        <div className={styles.body}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="provider-name">
              Name
            </label>
            <input
              id="provider-name"
              type="text"
              className={styles.input}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          {basicFields.map((f) => (
            <SchemaFormField
              key={f.name}
              schema={f}
              value={values[f.name]}
              onChange={(v) => setValues((s) => ({ ...s, [f.name]: v }))}
            />
          ))}

          {advancedFields.length > 0 && (
            <>
              <button
                type="button"
                className={styles.advancedToggle}
                onClick={() => setShowAdvanced((s) => !s)}
              >
                {showAdvanced ? 'Hide' : 'Show'} Advanced
              </button>
              {showAdvanced &&
                advancedFields.map((f) => (
                  <SchemaFormField
                    key={f.name}
                    schema={f}
                    value={values[f.name]}
                    onChange={(v) => setValues((s) => ({ ...s, [f.name]: v }))}
                  />
                ))}
            </>
          )}

          <div className={styles.extrasSection}>
            {renderExtras(extrasState, setExtrasState)}
          </div>
        </div>

        <footer className={styles.footer}>
          <button type="button" onClick={onCancel}>
            Cancel
          </button>
          <button
            type="button"
            className={styles.primary}
            disabled={!canSubmit}
            onClick={handleSubmit}
          >
            Save
          </button>
        </footer>
      </div>
    </div>
  )
}
