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
  isOpen,
  title,
  providers,
  onPick,
  onCancel,
}: ProviderPickerModalProps) {
  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, onCancel])

  if (!isOpen) return null

  return (
    <div className={styles.backdrop} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.header}>
          <h2>{title}</h2>
          <button aria-label="Close" className={styles.close} onClick={onCancel}>
            ×
          </button>
        </header>
        {providers.length === 0 ? (
          <p className={styles.empty}>No providers available.</p>
        ) : (
          <ul className={styles.list}>
            {providers.map((p) => (
              <li key={p.implementation}>
                <button
                  type="button"
                  className={styles.row}
                  onClick={() => onPick(p)}
                >
                  <div className={styles.name}>{p.name}</div>
                  <div className={styles.impl}>{p.implementation}</div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
