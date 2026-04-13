import { useState } from 'react'
import { useSystemStatus, useHealth, useRootFolders, useBackups } from '../api/hooks'
import type { HealthItem, SystemStatus, RootFolder, BackupInfo } from '../api/types'
import styles from './System.module.css'

type SystemTab = 'status' | 'diskspace' | 'tasks' | 'backups'

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(1)} ${units[i]}`
}

function healthItemClass(type: string): string {
  switch (type.toLowerCase()) {
    case 'error':
      return styles.healthItemError ?? ''
    case 'warning':
      return styles.healthItemWarning ?? ''
    default:
      return styles.healthItemNotice ?? ''
  }
}

function healthTypeClass(type: string): string {
  switch (type.toLowerCase()) {
    case 'error':
      return styles.healthTypeError ?? ''
    case 'warning':
      return styles.healthTypeWarning ?? ''
    default:
      return styles.healthTypeNotice ?? ''
  }
}

// ── Status card ───────────────────────────────────────────────────────────────

function StatusCard({ status }: { status: SystemStatus }) {
  const fields: { label: string; value: string; mono?: boolean }[] = [
    { label: 'Version', value: status.version },
    { label: 'App Name', value: status.appName },
    { label: 'Database', value: status.databaseType },
    { label: 'Runtime', value: status.runtimeName },
  ]

  // Augment with any extra fields that may be present in the response
  const extra = status as unknown as Record<string, unknown>
  if (typeof extra['startTime'] === 'string') {
    fields.push({ label: 'Started', value: formatDate(extra['startTime'] as string) })
  }
  if (typeof extra['osName'] === 'string') {
    fields.push({ label: 'OS', value: extra['osName'] as string })
  }
  if (typeof extra['osVersion'] === 'string') {
    fields.push({ label: 'OS Version', value: extra['osVersion'] as string })
  }
  if (typeof extra['isDocker'] === 'boolean') {
    fields.push({ label: 'Docker', value: extra['isDocker'] ? 'Yes' : 'No' })
  }

  return (
    <div className={styles.card}>
      <h2 className={styles.cardHeading}>Status</h2>
      <div className={styles.metaGrid}>
        {fields.map((f) => (
          <div key={f.label} className={styles.metaItem}>
            <span className={styles.metaLabel}>{f.label}</span>
            <span className={`${styles.metaValue} ${f.mono ? styles.metaMono : ''}`}>
              {f.value || '—'}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ── Health card ───────────────────────────────────────────────────────────────

function HealthCard({ items }: { items: HealthItem[] }) {
  return (
    <div className={styles.card}>
      <h2 className={styles.cardHeading}>Health</h2>
      {items.length === 0 ? (
        <p className={styles.successMessage}>
          <span className={styles.successDot} />
          All systems healthy
        </p>
      ) : (
        <ul className={styles.healthList}>
          {items.map((item, idx) => (
            <li
              key={idx}
              className={`${styles.healthItem} ${healthItemClass(item.type)}`}
            >
              <div>
                <span className={`${styles.healthType} ${healthTypeClass(item.type)}`}>
                  {item.type}
                </span>
              </div>
              <div>
                <p className={styles.healthMessage}>{item.message}</p>
                <p className={styles.healthSource}>{item.source}</p>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

// ── Disk Space card ──────────────────────────────────────────────────────────

function DiskSpaceCard({ folders }: { folders: RootFolder[] }) {
  return (
    <div className={styles.card}>
      <h2 className={styles.cardHeading}>Disk Space</h2>
      {folders.length === 0 ? (
        <p className={styles.stateMessage}>No root folders configured.</p>
      ) : (
        <div className={styles.diskList}>
          {folders.map((f) => (
            <div key={f.id} className={styles.diskItem}>
              <div className={styles.diskPath}>{f.path}</div>
              <div className={styles.diskInfo}>
                <span className={styles.diskFree}>
                  {f.freeSpace > 0 ? `${formatBytes(f.freeSpace)} free` : 'Unknown'}
                </span>
                <span className={f.accessible ? styles.diskAccessible : styles.diskInaccessible}>
                  {f.accessible ? 'Accessible' : 'Inaccessible'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Tasks card (placeholder) ─────────────────────────────────────────────────

function TasksCard() {
  return (
    <div className={styles.card}>
      <h2 className={styles.cardHeading}>Scheduled Tasks</h2>
      <p className={styles.stateMessage}>
        Task scheduling is not yet exposed via the API. Scheduled tasks will appear here
        once the scheduler endpoint is implemented.
      </p>
    </div>
  )
}

// ── Backups card ─────────────────────────────────────────────────────────────

function BackupsCard({ backups }: { backups: BackupInfo[] }) {
  return (
    <div className={styles.card}>
      <h2 className={styles.cardHeading}>Backups</h2>
      {backups.length === 0 ? (
        <p className={styles.stateMessage}>No backups found.</p>
      ) : (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Name</th>
                <th className={styles.th}>Size</th>
                <th className={styles.th}>Date</th>
              </tr>
            </thead>
            <tbody>
              {backups.map((b) => (
                <tr key={b.name} className={styles.tableRow}>
                  <td className={styles.td}>
                    <a
                      href={`/api/v6/system/backup/${b.name}`}
                      className={styles.backupLink}
                      download
                    >
                      {b.name}
                    </a>
                  </td>
                  <td className={styles.td}>{formatBytes(b.size)}</td>
                  <td className={styles.td}>{formatDate(b.time)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ── System page ───────────────────────────────────────────────────────────────

export function System() {
  const [activeTab, setActiveTab] = useState<SystemTab>('status')

  const { data: status, isLoading: loadingStatus, isError: errorStatus, error: statusErr } = useSystemStatus()
  const { data: health, isLoading: loadingHealth, isError: errorHealth, error: healthErr } = useHealth()
  const { data: rootFolders, isLoading: loadingFolders } = useRootFolders()
  const { data: backups, isLoading: loadingBackups } = useBackups()

  const healthItems: HealthItem[] = Array.isArray(health) ? health : []
  const folders: RootFolder[] = Array.isArray(rootFolders) ? rootFolders : []
  const backupList: BackupInfo[] = Array.isArray(backups) ? backups : []

  const tabs: { key: SystemTab; label: string }[] = [
    { key: 'status', label: 'Status' },
    { key: 'diskspace', label: 'Disk Space' },
    { key: 'tasks', label: 'Tasks' },
    { key: 'backups', label: 'Backups' },
  ]

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>System</h1>

      <div className={styles.tabBar}>
        {tabs.map((tab) => (
          <button
            key={tab.key}
            className={`${styles.tab} ${activeTab === tab.key ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'status' && (
        <>
          {loadingStatus && <p className={styles.stateMessage}>Loading system status...</p>}
          {errorStatus && (
            <p className={styles.errorMessage}>
              Failed to load status: {statusErr instanceof Error ? statusErr.message : 'Unknown error'}
            </p>
          )}
          {!loadingStatus && !errorStatus && status && <StatusCard status={status} />}

          {loadingHealth && <p className={styles.stateMessage}>Loading health checks...</p>}
          {errorHealth && (
            <p className={styles.errorMessage}>
              Failed to load health: {healthErr instanceof Error ? healthErr.message : 'Unknown error'}
            </p>
          )}
          {!loadingHealth && !errorHealth && <HealthCard items={healthItems} />}
        </>
      )}

      {activeTab === 'diskspace' && (
        <>
          {loadingFolders && <p className={styles.stateMessage}>Loading disk space...</p>}
          {!loadingFolders && <DiskSpaceCard folders={folders} />}
        </>
      )}

      {activeTab === 'tasks' && <TasksCard />}

      {activeTab === 'backups' && (
        <>
          {loadingBackups && <p className={styles.stateMessage}>Loading backups...</p>}
          {!loadingBackups && <BackupsCard backups={backupList} />}
        </>
      )}
    </div>
  )
}
