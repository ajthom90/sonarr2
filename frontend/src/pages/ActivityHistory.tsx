import { useState } from 'react'
import { useHistory } from '../api/hooks'
import type { HistoryEntry } from '../api/types'
import styles from './Activity.module.css'

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

function dedupeById<T extends { id: number }>(items: T[]): T[] {
  const seen = new Set<number>()
  return items.filter((item) => {
    if (seen.has(item.id)) return false
    seen.add(item.id)
    return true
  })
}

/**
 * ActivityHistory is the /activity/history page. Focused on history only
 * (no queue tab) — Queue lives at /activity/queue and will get its own
 * page once the backend /api/v3/queue endpoint lands.
 */
export function ActivityHistory() {
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [allEntries, setAllEntries] = useState<HistoryEntry[]>([])

  const { data, isLoading, isError, error } = useHistory(cursor)
  const currentPageEntries: HistoryEntry[] = data?.data ?? []
  const combined = dedupeById([...allEntries, ...currentPageEntries])

  const hasMore = data?.pagination.hasMore ?? false
  const nextCursor = data?.pagination.nextCursor

  function loadMore() {
    if (nextCursor) {
      setAllEntries(combined)
      setCursor(nextCursor)
    }
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>History</h1>

      {isLoading && combined.length === 0 && (
        <p className={styles.stateMessage}>Loading history…</p>
      )}
      {isError && combined.length === 0 && (
        <p className={styles.errorMessage}>
          Failed to load history:{' '}
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      )}
      {!isLoading && combined.length === 0 && !isError && (
        <p className={styles.stateMessage}>
          No history yet. Grab / import / delete events show up here as they happen.
        </p>
      )}

      {combined.length > 0 && (
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
                    <span
                      className={`${styles.badge} ${historyBadgeClass(entry.eventType)}`}
                    >
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
                {isLoading ? 'Loading…' : 'Load more'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
