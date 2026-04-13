import { useState } from 'react'
import { useCommands, useHistory } from '../api/hooks'
import type { Command, HistoryEntry } from '../api/types'
import styles from './Activity.module.css'

type Tab = 'queue' | 'history'

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

function commandBadgeClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'queued':
      return styles.badgeQueued ?? ''
    case 'started':
    case 'running':
      return styles.badgeRunning ?? ''
    case 'completed':
      return styles.badgeCompleted ?? ''
    case 'failed':
      return styles.badgeFailed ?? ''
    default:
      return styles.badgeDefault ?? ''
  }
}

function historyBadgeClass(eventType: string): string {
  switch (eventType.toLowerCase()) {
    case 'grabbed':
      return styles.badgeQueued ?? ''
    case 'downloadfolderimported':
    case 'downloadimported':
    case 'imported':
      return styles.badgeCompleted ?? ''
    case 'downloadfailed':
    case 'failed':
      return styles.badgeFailed ?? ''
    default:
      return styles.badgeDefault ?? ''
  }
}

// ── Queue tab ─────────────────────────────────────────────────────────────────

function QueueTab() {
  const { data, isLoading, isError, error } = useCommands()
  const commands: Command[] = data?.data ?? []

  if (isLoading) return <p className={styles.stateMessage}>Loading commands...</p>
  if (isError)
    return (
      <p className={styles.errorMessage}>
        Failed to load queue: {error instanceof Error ? error.message : 'Unknown error'}
      </p>
    )
  if (commands.length === 0)
    return <p className={styles.stateMessage}>No recent commands.</p>

  return (
    <div className={styles.tableWrapper}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th className={styles.th}>Command</th>
            <th className={styles.th}>Status</th>
            <th className={styles.th}>Queued</th>
            <th className={styles.th}>Started</th>
            <th className={styles.th}>Ended</th>
          </tr>
        </thead>
        <tbody>
          {commands.map((cmd) => (
            <tr key={cmd.id} className={styles.row}>
              <td className={styles.td}>{cmd.name}</td>
              <td className={styles.td}>
                <span className={`${styles.badge} ${commandBadgeClass(cmd.status)}`}>
                  {cmd.status}
                </span>
              </td>
              <td className={styles.td}>
                <span className={styles.muted}>{formatDate(cmd.queued)}</span>
              </td>
              <td className={styles.td}>
                <span className={styles.muted}>{cmd.started ? formatDate(cmd.started) : '—'}</span>
              </td>
              <td className={styles.td}>
                <span className={styles.muted}>{cmd.ended ? formatDate(cmd.ended) : '—'}</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ── History tab ───────────────────────────────────────────────────────────────

function HistoryTab() {
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [allEntries, setAllEntries] = useState<HistoryEntry[]>([])

  const { data, isLoading, isError, error } = useHistory(cursor)

  // Merge new page into allEntries when data changes
  const currentPageEntries: HistoryEntry[] = data?.data ?? []

  // Build the deduplicated combined list: previously loaded + current page
  const combined = dedupeById([...allEntries, ...currentPageEntries])

  const hasMore = data?.pagination.hasMore ?? false
  const nextCursor = data?.pagination.nextCursor

  function loadMore() {
    if (nextCursor) {
      setAllEntries(combined)
      setCursor(nextCursor)
    }
  }

  if (isLoading && combined.length === 0)
    return <p className={styles.stateMessage}>Loading history...</p>

  if (isError && combined.length === 0)
    return (
      <p className={styles.errorMessage}>
        Failed to load history: {error instanceof Error ? error.message : 'Unknown error'}
      </p>
    )

  if (combined.length === 0)
    return <p className={styles.stateMessage}>No history yet.</p>

  return (
    <>
      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.th}>Date</th>
              <th className={styles.th}>Event</th>
              <th className={styles.th}>Source Title</th>
              <th className={styles.th}>Series</th>
              <th className={styles.th}>Episode</th>
            </tr>
          </thead>
          <tbody>
            {combined.map((entry) => (
              <tr key={entry.id} className={styles.row}>
                <td className={styles.td}>
                  <span className={styles.muted}>{formatDate(entry.date)}</span>
                </td>
                <td className={styles.td}>
                  <span className={`${styles.badge} ${historyBadgeClass(entry.eventType)}`}>
                    {entry.eventType}
                  </span>
                </td>
                <td className={styles.td}>
                  <span className={styles.sourceTitle} title={entry.sourceTitle}>
                    {entry.sourceTitle || <span className={styles.muted}>—</span>}
                  </span>
                </td>
                <td className={styles.td}>
                  <span className={styles.muted}>#{entry.seriesId}</span>
                </td>
                <td className={styles.td}>
                  <span className={styles.muted}>#{entry.episodeId}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {(hasMore || isLoading) && (
          <div className={styles.loadMoreWrapper}>
            <button
              className={styles.loadMoreBtn}
              onClick={loadMore}
              disabled={isLoading}
            >
              {isLoading ? 'Loading...' : 'Load more'}
            </button>
          </div>
        )}
      </div>
    </>
  )
}

function dedupeById<T extends { id: number }>(items: T[]): T[] {
  const seen = new Set<number>()
  return items.filter((item) => {
    if (seen.has(item.id)) return false
    seen.add(item.id)
    return true
  })
}

// ── Activity page ─────────────────────────────────────────────────────────────

export function Activity() {
  const [activeTab, setActiveTab] = useState<Tab>('queue')

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Activity</h1>

      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${activeTab === 'queue' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('queue')}
        >
          Queue
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'history' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('history')}
        >
          History
        </button>
      </div>

      {activeTab === 'queue' && <QueueTab />}
      {activeTab === 'history' && <HistoryTab />}
    </div>
  )
}
