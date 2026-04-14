import { useEffect, useState } from 'react'
import { useSeriesLookup } from '../api/hooks'
import type { SeriesLookupResult } from '../api/types'
import styles from './SearchOverrideModal.module.css'

export interface SearchOverrideModalProps {
  isOpen: boolean
  initialTerm: string
  onSelect: (match: SeriesLookupResult | null) => void
  onCancel: () => void
}

export function SearchOverrideModal({
  isOpen,
  initialTerm,
  onSelect,
  onCancel,
}: SearchOverrideModalProps) {
  const [term, setTerm] = useState(initialTerm)
  const [debounced, setDebounced] = useState(initialTerm)

  useEffect(() => {
    const id = setTimeout(() => setDebounced(term), 300)
    return () => clearTimeout(id)
  }, [term])

  const { data, isLoading } = useSeriesLookup(debounced)

  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, onCancel])

  if (!isOpen) return null

  const results = data ?? []

  return (
    <div className={styles.backdrop} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.header}>
          <h2>Search TVDB</h2>
          <button aria-label="Close" className={styles.close} onClick={onCancel}>×</button>
        </header>

        <input
          className={styles.input}
          autoFocus
          type="text"
          placeholder="Series title"
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />

        {isLoading && <p className={styles.status}>Searching…</p>}

        <ul className={styles.list}>
          {results.map((r) => (
            <li key={r.tvdbId}>
              <button type="button" className={styles.row} onClick={() => onSelect(r)}>
                <div className={styles.title}>
                  {r.title} <span className={styles.year}>({r.year})</span>
                </div>
                {r.overview && <div className={styles.overview}>{r.overview}</div>}
              </button>
            </li>
          ))}
          {results.length === 0 && !isLoading && debounced.length >= 2 && (
            <li>
              <p className={styles.status}>No matches for &ldquo;{debounced}&rdquo;.</p>
            </li>
          )}
        </ul>

        <footer className={styles.footer}>
          <button type="button" onClick={() => onSelect(null)}>Clear match</button>
          <button type="button" onClick={onCancel}>Cancel</button>
        </footer>
      </div>
    </div>
  )
}
