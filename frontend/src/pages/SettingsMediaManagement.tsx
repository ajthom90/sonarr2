import { useState } from 'react'
import { useRootFolders, useCreateRootFolder, useDeleteRootFolder } from '../api/hooks'
import type { RootFolder } from '../api/types'
import { FileBrowserModal } from '../components/FileBrowserModal'
import styles from './SettingsMediaManagement.module.css'

// ── localStorage keys for naming config (backend not yet available) ──────────

const STORAGE_KEY = 'sonarr2_naming_config'

interface NamingConfig {
  renameEpisodes: boolean
  standardFormat: string
  dailyFormat: string
  animeFormat: string
  seasonFolderFormat: string
}

const defaultNaming: NamingConfig = {
  renameEpisodes: true,
  standardFormat: '{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}',
  dailyFormat: '{Series Title} - {Air-Date} - {Episode Title} {Quality Full}',
  animeFormat: '{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}',
  seasonFolderFormat: 'Season {season:00}',
}

function loadNaming(): NamingConfig {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...defaultNaming, ...JSON.parse(raw) }
  } catch {
    // ignore
  }
  return { ...defaultNaming }
}

function saveNaming(config: NamingConfig) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const value = bytes / Math.pow(1024, i)
  return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

// ── Episode Naming Section ──────────────────────────────────────────────────

function EpisodeNamingSection() {
  const [config, setConfig] = useState<NamingConfig>(loadNaming)
  const [saved, setSaved] = useState(false)

  function handleSave() {
    saveNaming(config)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Episode Naming</h2>

      <div className={styles.toggleRow}>
        <input
          type="checkbox"
          id="renameEpisodes"
          className={styles.toggle}
          checked={config.renameEpisodes}
          onChange={(e) => setConfig((c) => ({ ...c, renameEpisodes: e.target.checked }))}
        />
        <label htmlFor="renameEpisodes" className={styles.toggleLabel}>
          Rename Episodes
        </label>
      </div>

      <label className={styles.label}>
        <span className={styles.labelText}>Standard Episode Format</span>
        <input
          type="text"
          className={styles.input}
          value={config.standardFormat}
          onChange={(e) => setConfig((c) => ({ ...c, standardFormat: e.target.value }))}
        />
        <span className={styles.hint}>
          Used for standard series (e.g. S01E01)
        </span>
      </label>

      <label className={styles.label}>
        <span className={styles.labelText}>Daily Episode Format</span>
        <input
          type="text"
          className={styles.input}
          value={config.dailyFormat}
          onChange={(e) => setConfig((c) => ({ ...c, dailyFormat: e.target.value }))}
        />
        <span className={styles.hint}>
          Used for daily series (e.g. talk shows)
        </span>
      </label>

      <label className={styles.label}>
        <span className={styles.labelText}>Anime Episode Format</span>
        <input
          type="text"
          className={styles.input}
          value={config.animeFormat}
          onChange={(e) => setConfig((c) => ({ ...c, animeFormat: e.target.value }))}
        />
        <span className={styles.hint}>
          Used for anime series
        </span>
      </label>

      <label className={styles.label}>
        <span className={styles.labelText}>Season Folder Format</span>
        <input
          type="text"
          className={styles.input}
          value={config.seasonFolderFormat}
          onChange={(e) => setConfig((c) => ({ ...c, seasonFolderFormat: e.target.value }))}
        />
        <span className={styles.hint}>
          Format for season folders within the series folder
        </span>
      </label>

      <div className={styles.saveBar}>
        <button className={styles.saveBtn} onClick={handleSave}>
          Save
        </button>
        {saved && <span className={styles.savedMessage}>Saved to local storage</span>}
      </div>
    </section>
  )
}

// ── Root Folders Section ────────────────────────────────────────────────────

function RootFoldersSection() {
  const { data: rootFolders, isLoading, isError, error } = useRootFolders()
  const createRF = useCreateRootFolder()
  const deleteRF = useDeleteRootFolder()
  const [browserOpen, setBrowserOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const folders: RootFolder[] = rootFolders ?? []

  async function handleSelect(path: string) {
    setBrowserOpen(false)
    try {
      await createRF.mutateAsync({ path })
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleDelete(rf: RootFolder) {
    if (!confirm(`Remove ${rf.path}? This will not delete any files on disk.`)) return
    try {
      await deleteRF.mutateAsync(rf.id)
      setDeleteError(null)
    } catch (err) {
      setDeleteError((err as Error).message)
    }
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Root Folders</h2>

      {isLoading && <p className={styles.stateMessage}>Loading root folders…</p>}
      {isError && (
        <p className={styles.errorMessage}>
          Failed to load root folders: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}
      {!isLoading && !isError && folders.length === 0 && (
        <p className={styles.stateMessage}>No root folders yet.</p>
      )}

      {folders.length > 0 && (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Path</th>
                <th className={styles.th}>Free Space</th>
                <th className={styles.th}>Accessible</th>
                <th className={styles.th}></th>
              </tr>
            </thead>
            <tbody>
              {folders.map((folder) => (
                <tr key={folder.id} className={styles.row}>
                  <td className={styles.td}>{folder.path}</td>
                  <td className={styles.td}>
                    <span className={styles.muted}>{formatBytes(folder.freeSpace)}</span>
                  </td>
                  <td className={styles.td}>
                    <span className={`${styles.pill} ${folder.accessible ? styles.pillEnabled : styles.pillDisabled}`}>
                      {folder.accessible ? 'Yes' : 'No'}
                    </span>
                  </td>
                  <td className={styles.td}>
                    <button className={styles.deleteBtn} onClick={() => handleDelete(folder)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className={styles.addRow}>
        <button className={styles.addBtn} onClick={() => setBrowserOpen(true)}>
          Add Root Folder
        </button>
      </div>

      <FileBrowserModal
        isOpen={browserOpen}
        initialPath="/"
        onSelect={handleSelect}
        onCancel={() => setBrowserOpen(false)}
      />

      {deleteError && (
        <div className={styles.errorMessage}>
          {deleteError}
          <button
            onClick={() => setDeleteError(null)}
            style={{ marginLeft: 8 }}
          >
            Dismiss
          </button>
        </div>
      )}
    </section>
  )
}

// ── Main page ───────────────────────────────────────────────────────────────

export function SettingsMediaManagement() {
  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Media Management</h1>
      <EpisodeNamingSection />
      <RootFoldersSection />
    </div>
  )
}
