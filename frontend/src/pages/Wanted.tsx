import { useState } from 'react'
import { useWantedMissing } from '../api/hooks'
import type { WantedEpisode } from '../api/types'
import styles from './Wanted.module.css'

function formatDate(dateStr?: string): string {
  if (!dateStr) return '—'
  try {
    return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return dateStr
  }
}

function formatEpCode(season: number, episode: number): string {
  return `S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
}

function dedupeById(items: WantedEpisode[]): WantedEpisode[] {
  const seen = new Set<number>()
  return items.filter((item) => {
    if (seen.has(item.id)) return false
    seen.add(item.id)
    return true
  })
}

export function Wanted() {
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [prevEntries, setPrevEntries] = useState<WantedEpisode[]>([])

  const { data, isLoading, isError, error } = useWantedMissing(cursor)

  const currentPage: WantedEpisode[] = data?.data ?? []
  const combined = dedupeById([...prevEntries, ...currentPage])

  const hasMore = data?.pagination.hasMore ?? false
  const nextCursor = data?.pagination.nextCursor

  function loadMore() {
    if (nextCursor) {
      setPrevEntries(combined)
      setCursor(nextCursor)
    }
  }

  if (isLoading && combined.length === 0) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Wanted</h1>
        <p className={styles.stateMessage}>Loading missing episodes...</p>
      </div>
    )
  }

  if (isError && combined.length === 0) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Wanted</h1>
        <p className={styles.errorMessage}>
          Failed to load wanted list: {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      </div>
    )
  }

  if (!isLoading && combined.length === 0) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Wanted</h1>
        <p className={styles.stateMessage}>
          <span className={styles.emptyIcon}>&#10003;</span>
          No missing episodes — all monitored episodes have files.
        </p>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Wanted</h1>

      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.th}>Series</th>
              <th className={styles.th}>Episode</th>
              <th className={styles.th}>Title</th>
              <th className={styles.th}>Air Date</th>
            </tr>
          </thead>
          <tbody>
            {combined.map((ep) => (
              <tr key={ep.id} className={styles.row}>
                <td className={styles.td}>
                  {ep.series?.title ? (
                    ep.series.title
                  ) : (
                    <span className={styles.muted}>Series #{ep.seriesId}</span>
                  )}
                </td>
                <td className={styles.td}>
                  <span className={styles.epCode}>
                    {formatEpCode(ep.seasonNumber, ep.episodeNumber)}
                  </span>
                </td>
                <td className={styles.td}>{ep.title || <span className={styles.muted}>TBA</span>}</td>
                <td className={styles.td}>
                  <span className={styles.muted}>{formatDate(ep.airDate)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {(hasMore || isLoading) && (
          <div className={styles.loadMoreWrapper}>
            <button className={styles.loadMoreBtn} onClick={loadMore} disabled={isLoading}>
              {isLoading ? 'Loading...' : 'Load more'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
