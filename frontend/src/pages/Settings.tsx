import { useState } from 'react'
import {
  useIndexers,
  useDownloadClients,
  useQualityProfiles,
  useDeleteIndexer,
  useAddIndexer,
  useDeleteDownloadClient,
  useAddDownloadClient,
} from '../api/hooks'
import type { Indexer, DownloadClient, QualityProfile } from '../api/types'
import styles from './Settings.module.css'

type Tab = 'indexers' | 'downloadclients' | 'qualityprofiles'

// ── Helpers ───────────────────────────────────────────────────────────────────

function parseSettingsJson(raw: string): Record<string, unknown> {
  if (!raw.trim()) return {}
  return JSON.parse(raw)
}

// ── Indexers tab ──────────────────────────────────────────────────────────────

interface IndexerFormValues {
  name: string
  implementation: string
  settings: string
}

const emptyIndexerForm: IndexerFormValues = { name: '', implementation: '', settings: '' }

function IndexersTab() {
  const { data, isLoading, isError, error } = useIndexers()
  const deleteMutation = useDeleteIndexer()
  const addMutation = useAddIndexer()

  const indexers: Indexer[] = data?.data ?? []
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<IndexerFormValues>(emptyIndexerForm)
  const [formError, setFormError] = useState<string | null>(null)

  function handleDelete(id: number) {
    deleteMutation.mutate(id)
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setFormError(null)
    let parsedSettings: Record<string, unknown> = {}
    try {
      parsedSettings = parseSettingsJson(form.settings)
    } catch {
      setFormError('Settings JSON is invalid')
      return
    }
    const body = {
      name: form.name.trim(),
      implementation: form.implementation.trim(),
      ...parsedSettings,
    }
    addMutation.mutate(body, {
      onSuccess: () => {
        setForm(emptyIndexerForm)
        setShowForm(false)
      },
      onError: (err) => {
        setFormError(err instanceof Error ? err.message : 'Failed to add indexer')
      },
    })
  }

  if (isLoading) return <p className={styles.stateMessage}>Loading indexers...</p>
  if (isError)
    return (
      <p className={styles.errorMessage}>
        Failed to load indexers: {error instanceof Error ? error.message : 'Unknown error'}
      </p>
    )

  return (
    <div>
      <div className={styles.sectionHeader}>
        <h2 className={styles.sectionTitle}>Indexers</h2>
        <button className={styles.addBtn} onClick={() => { setShowForm(v => !v); setFormError(null) }}>
          {showForm ? 'Cancel' : '+ Add Indexer'}
        </button>
      </div>

      {showForm && (
        <div className={styles.formCard}>
          <p className={styles.formTitle}>New Indexer</p>
          <form onSubmit={handleSubmit}>
            <div className={styles.formRow}>
              <div className={styles.formGroup}>
                <label className={styles.formLabel}>Name</label>
                <input
                  className={styles.formInput}
                  type="text"
                  required
                  placeholder="My Indexer"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                />
              </div>
              <div className={styles.formGroup}>
                <label className={styles.formLabel}>Implementation</label>
                <input
                  className={styles.formInput}
                  type="text"
                  required
                  placeholder="Newznab"
                  value={form.implementation}
                  onChange={e => setForm(f => ({ ...f, implementation: e.target.value }))}
                />
              </div>
            </div>
            <div className={styles.formGroup} style={{ marginBottom: 'var(--space-4)' }}>
              <label className={styles.formLabel}>Settings (JSON)</label>
              <textarea
                className={styles.formTextarea}
                placeholder='{"baseUrl": "https://...", "apiKey": "..."}'
                value={form.settings}
                onChange={e => setForm(f => ({ ...f, settings: e.target.value }))}
              />
            </div>
            {formError && <p className={styles.errorText}>{formError}</p>}
            <div className={styles.formActions}>
              <button className={styles.submitBtn} type="submit" disabled={addMutation.isPending}>
                {addMutation.isPending ? 'Saving...' : 'Save'}
              </button>
              <button
                className={styles.cancelBtn}
                type="button"
                onClick={() => { setShowForm(false); setForm(emptyIndexerForm); setFormError(null) }}
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {indexers.length === 0 ? (
        <p className={styles.stateMessage}>No indexers configured.</p>
      ) : (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Name</th>
                <th className={styles.th}>Implementation</th>
                <th className={styles.th}>RSS</th>
                <th className={styles.th}>Search</th>
                <th className={styles.th}>Priority</th>
                <th className={styles.th}></th>
              </tr>
            </thead>
            <tbody>
              {indexers.map(indexer => (
                <tr key={indexer.id} className={styles.row}>
                  <td className={styles.td}>{indexer.name}</td>
                  <td className={styles.td}><span className={styles.muted}>{indexer.implementation}</span></td>
                  <td className={styles.td}>
                    <span className={`${styles.pill} ${indexer.enableRss ? styles.pillEnabled : styles.pillDisabled}`}>
                      {indexer.enableRss ? 'Yes' : 'No'}
                    </span>
                  </td>
                  <td className={styles.td}>
                    <span className={`${styles.pill} ${indexer.enableAutomaticSearch ? styles.pillEnabled : styles.pillDisabled}`}>
                      {indexer.enableAutomaticSearch ? 'Yes' : 'No'}
                    </span>
                  </td>
                  <td className={styles.td}><span className={styles.muted}>{indexer.priority}</span></td>
                  <td className={styles.td}>
                    <button
                      className={styles.deleteBtn}
                      onClick={() => handleDelete(indexer.id)}
                      disabled={deleteMutation.isPending}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ── Download Clients tab ──────────────────────────────────────────────────────

interface DownloadClientFormValues {
  name: string
  implementation: string
  settings: string
}

const emptyDCForm: DownloadClientFormValues = { name: '', implementation: '', settings: '' }

function DownloadClientsTab() {
  const { data, isLoading, isError, error } = useDownloadClients()
  const deleteMutation = useDeleteDownloadClient()
  const addMutation = useAddDownloadClient()

  const clients: DownloadClient[] = data?.data ?? []
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<DownloadClientFormValues>(emptyDCForm)
  const [formError, setFormError] = useState<string | null>(null)

  function handleDelete(id: number) {
    deleteMutation.mutate(id)
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setFormError(null)
    let parsedSettings: Record<string, unknown> = {}
    try {
      parsedSettings = parseSettingsJson(form.settings)
    } catch {
      setFormError('Settings JSON is invalid')
      return
    }
    const body = {
      name: form.name.trim(),
      implementation: form.implementation.trim(),
      ...parsedSettings,
    }
    addMutation.mutate(body, {
      onSuccess: () => {
        setForm(emptyDCForm)
        setShowForm(false)
      },
      onError: (err) => {
        setFormError(err instanceof Error ? err.message : 'Failed to add download client')
      },
    })
  }

  if (isLoading) return <p className={styles.stateMessage}>Loading download clients...</p>
  if (isError)
    return (
      <p className={styles.errorMessage}>
        Failed to load download clients: {error instanceof Error ? error.message : 'Unknown error'}
      </p>
    )

  return (
    <div>
      <div className={styles.sectionHeader}>
        <h2 className={styles.sectionTitle}>Download Clients</h2>
        <button className={styles.addBtn} onClick={() => { setShowForm(v => !v); setFormError(null) }}>
          {showForm ? 'Cancel' : '+ Add Download Client'}
        </button>
      </div>

      {showForm && (
        <div className={styles.formCard}>
          <p className={styles.formTitle}>New Download Client</p>
          <form onSubmit={handleSubmit}>
            <div className={styles.formRow}>
              <div className={styles.formGroup}>
                <label className={styles.formLabel}>Name</label>
                <input
                  className={styles.formInput}
                  type="text"
                  required
                  placeholder="SABnzbd"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                />
              </div>
              <div className={styles.formGroup}>
                <label className={styles.formLabel}>Implementation</label>
                <input
                  className={styles.formInput}
                  type="text"
                  required
                  placeholder="Sabnzbd"
                  value={form.implementation}
                  onChange={e => setForm(f => ({ ...f, implementation: e.target.value }))}
                />
              </div>
            </div>
            <div className={styles.formGroup} style={{ marginBottom: 'var(--space-4)' }}>
              <label className={styles.formLabel}>Settings (JSON)</label>
              <textarea
                className={styles.formTextarea}
                placeholder='{"host": "localhost", "port": 8080, "apiKey": "..."}'
                value={form.settings}
                onChange={e => setForm(f => ({ ...f, settings: e.target.value }))}
              />
            </div>
            {formError && <p className={styles.errorText}>{formError}</p>}
            <div className={styles.formActions}>
              <button className={styles.submitBtn} type="submit" disabled={addMutation.isPending}>
                {addMutation.isPending ? 'Saving...' : 'Save'}
              </button>
              <button
                className={styles.cancelBtn}
                type="button"
                onClick={() => { setShowForm(false); setForm(emptyDCForm); setFormError(null) }}
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {clients.length === 0 ? (
        <p className={styles.stateMessage}>No download clients configured.</p>
      ) : (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Name</th>
                <th className={styles.th}>Implementation</th>
                <th className={styles.th}>Enabled</th>
                <th className={styles.th}>Priority</th>
                <th className={styles.th}></th>
              </tr>
            </thead>
            <tbody>
              {clients.map(client => (
                <tr key={client.id} className={styles.row}>
                  <td className={styles.td}>{client.name}</td>
                  <td className={styles.td}><span className={styles.muted}>{client.implementation}</span></td>
                  <td className={styles.td}>
                    <span className={`${styles.pill} ${client.enable ? styles.pillEnabled : styles.pillDisabled}`}>
                      {client.enable ? 'Enabled' : 'Disabled'}
                    </span>
                  </td>
                  <td className={styles.td}><span className={styles.muted}>{client.priority}</span></td>
                  <td className={styles.td}>
                    <button
                      className={styles.deleteBtn}
                      onClick={() => handleDelete(client.id)}
                      disabled={deleteMutation.isPending}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ── Quality Profiles tab ──────────────────────────────────────────────────────

function QualityProfilesTab() {
  const { data, isLoading, isError, error } = useQualityProfiles()
  const profiles: QualityProfile[] = data?.data ?? []

  if (isLoading) return <p className={styles.stateMessage}>Loading quality profiles...</p>
  if (isError)
    return (
      <p className={styles.errorMessage}>
        Failed to load quality profiles: {error instanceof Error ? error.message : 'Unknown error'}
      </p>
    )

  if (profiles.length === 0)
    return (
      <div>
        <div className={styles.sectionHeader}>
          <h2 className={styles.sectionTitle}>Quality Profiles</h2>
        </div>
        <p className={styles.stateMessage}>No quality profiles found.</p>
      </div>
    )

  return (
    <div>
      <div className={styles.sectionHeader}>
        <h2 className={styles.sectionTitle}>Quality Profiles</h2>
      </div>
      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.th}>Name</th>
              <th className={styles.th}>Upgrades Allowed</th>
              <th className={styles.th}>Quality Items</th>
            </tr>
          </thead>
          <tbody>
            {profiles.map(profile => (
              <tr key={profile.id} className={styles.row}>
                <td className={styles.td}>{profile.name}</td>
                <td className={styles.td}>
                  <span className={`${styles.pill} ${profile.upgradeAllowed ? styles.pillEnabled : styles.pillDisabled}`}>
                    {profile.upgradeAllowed ? 'Yes' : 'No'}
                  </span>
                </td>
                <td className={styles.td}>
                  <span className={styles.muted}>{profile.items?.length ?? 0}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ── Settings page ─────────────────────────────────────────────────────────────

export function Settings() {
  const [activeTab, setActiveTab] = useState<Tab>('indexers')

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Settings</h1>

      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${activeTab === 'indexers' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('indexers')}
        >
          Indexers
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'downloadclients' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('downloadclients')}
        >
          Download Clients
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'qualityprofiles' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('qualityprofiles')}
        >
          Quality Profiles
        </button>
      </div>

      {activeTab === 'indexers' && <IndexersTab />}
      {activeTab === 'downloadclients' && <DownloadClientsTab />}
      {activeTab === 'qualityprofiles' && <QualityProfilesTab />}
    </div>
  )
}
