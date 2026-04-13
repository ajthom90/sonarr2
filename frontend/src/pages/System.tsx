import { useSystemStatus, useHealth } from '../api/hooks'
import type { HealthItem, SystemStatus } from '../api/types'
import styles from './System.module.css'

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

// ── System page ───────────────────────────────────────────────────────────────

export function System() {
  const { data: status, isLoading: loadingStatus, isError: errorStatus, error: statusErr } = useSystemStatus()
  const { data: health, isLoading: loadingHealth, isError: errorHealth, error: healthErr } = useHealth()

  const healthItems: HealthItem[] = Array.isArray(health) ? health : []

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>System</h1>

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
    </div>
  )
}
