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
          <input
            id={id}
            type="text"
            className={styles.input}
            placeholder={schema.placeholder}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)}
          />
        )
      case 'password':
        return (
          <input
            id={id}
            type="password"
            className={styles.input}
            placeholder={schema.placeholder}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)}
          />
        )
      case 'number':
        return (
          <input
            id={id}
            type="number"
            className={styles.input}
            placeholder={schema.placeholder}
            value={value === undefined || value === null ? '' : (value as number)}
            onChange={(e) =>
              onChange(e.target.value === '' ? null : Number(e.target.value))
            }
          />
        )
      case 'checkbox':
        return (
          <input
            id={id}
            type="checkbox"
            className={styles.checkbox}
            checked={Boolean(value)}
            onChange={(e) => onChange(e.target.checked)}
          />
        )
      case 'select':
      case 'multiselect':
        // Backend FieldSchema doesn't emit options yet. Fall back to text.
        // TODO: when backend adds options[], render real <select>.
        return (
          <input
            id={id}
            type="text"
            className={styles.input}
            placeholder={schema.placeholder ?? '(enter value)'}
            value={(value as string) ?? ''}
            onChange={(e) => onChange(e.target.value)}
          />
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
