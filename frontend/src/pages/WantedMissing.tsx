import { useState } from 'react'
import { useWantedMissingV3 } from '../api/hooks'
import type { WantedEpisodeV3 } from '../api/types'
import styles from './WantedTable.module.css'

function formatDate(iso?: string): string {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return iso
  }
}

function formatEpCode(season: number, episode: number): string {
  return `S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
}

/**
 * WantedMissing lists monitored episodes that have no file.
 * Backed by GET /api/v3/wanted/missing (paged).
 */
export function WantedMissing() {
  const [page, setPage] = useState(1)
  const pageSize = 50
  const { data, isLoading, isError, error } = useWantedMissingV3(page, pageSize)

  const records: WantedEpisodeV3[] = data?.records ?? []
  const total = data?.totalRecords ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Missing Episodes</h1>

      {isLoading && <p className={styles.status}>Loading missing episodes…</p>}
      {isError && (
        <p className={styles.error}>
          Failed to load: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}

      {!isLoading && records.length === 0 && !isError && (
        <p className={styles.status}>
          No missing episodes. All monitored episodes have files.
        </p>
      )}

      {records.length > 0 && (
        <>
          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th className={styles.th}>Series</th>
                  <th className={styles.th}>Episode</th>
                  <th className={styles.th}>Title</th>
                  <th className={styles.th}>Air Date</th>
                  <th className={styles.th}>Status</th>
                </tr>
              </thead>
              <tbody>
                {records.map((ep) => (
                  <tr key={ep.id} className={styles.row}>
                    <td className={styles.td}>#{ep.seriesId}</td>
                    <td className={styles.td}>
                      <code>{formatEpCode(ep.seasonNumber, ep.episodeNumber)}</code>
                    </td>
                    <td className={styles.td}>{ep.title}</td>
                    <td className={styles.td}>{formatDate(ep.airDateUtc)}</td>
                    <td className={styles.td}>
                      <span className={styles.pillMissing}>Missing</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={styles.pager}>
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              ← Prev
            </button>
            <span className={styles.pagerInfo}>
              Page {page} of {totalPages} · {total} episode{total === 1 ? '' : 's'}
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            >
              Next →
            </button>
          </div>
        </>
      )}
    </div>
  )
}
