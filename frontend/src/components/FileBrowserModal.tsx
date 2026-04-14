import { useEffect, useState } from 'react'
import { useFilesystem } from '../api/hooks'
import styles from './FileBrowserModal.module.css'

export interface FileBrowserModalProps {
  isOpen: boolean
  initialPath?: string
  title?: string
  onSelect: (path: string) => void
  onCancel: () => void
}

export function FileBrowserModal({
  isOpen,
  initialPath = '/',
  title = 'File Browser',
  onSelect,
  onCancel,
}: FileBrowserModalProps) {
  const [currentPath, setCurrentPath] = useState(initialPath)
  const { data, isLoading, error } = useFilesystem(currentPath)

  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
      if (e.key === 'Enter' && currentPath !== '/') onSelect(currentPath)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, currentPath, onCancel, onSelect])

  if (!isOpen) return null

  const parent = currentPath === '/' ? null : currentPath.replace(/\/[^/]+$/, '') || '/'
  const breadcrumbs = currentPath.split('/').filter(Boolean)
  const atRoot = currentPath === '/'

  return (
    <div className={styles.backdrop} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.header}>
          <h2>{title}</h2>
          <button className={styles.close} aria-label="Close" onClick={onCancel}>×</button>
        </header>

        <nav className={styles.breadcrumbs}>
          <button type="button" onClick={() => setCurrentPath('/')}>/</button>
          {breadcrumbs.map((seg, i) => {
            const to = '/' + breadcrumbs.slice(0, i + 1).join('/')
            return (
              <span key={to}>
                <span className={styles.sep}>/</span>
                <button type="button" onClick={() => setCurrentPath(to)}>{seg}</button>
              </span>
            )
          })}
        </nav>

        {isLoading && <p className={styles.status}>Loading…</p>}
        {error && <p className={styles.error}>{(error as Error).message}</p>}

        {data && (
          <ul className={styles.list}>
            {parent !== null && (
              <li>
                <button type="button" className={styles.row} onClick={() => setCurrentPath(parent)}>
                  <span className={styles.icon}>📁</span>
                  <span>..</span>
                </button>
              </li>
            )}
            {data.directories.map((d) => (
              <li key={d.path}>
                <button type="button" className={styles.row} onClick={() => setCurrentPath(d.path)}>
                  <span className={styles.icon}>📁</span>
                  <span>{d.name}</span>
                </button>
              </li>
            ))}
          </ul>
        )}

        <footer className={styles.footer}>
          <button type="button" onClick={onCancel}>Cancel</button>
          <button
            type="button"
            className={styles.primary}
            disabled={atRoot}
            onClick={() => onSelect(currentPath)}
          >
            OK
          </button>
        </footer>
      </div>
    </div>
  )
}
