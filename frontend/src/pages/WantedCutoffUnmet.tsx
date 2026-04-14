import { useState } from 'react'
import { useWantedCutoffUnmet } from '../api/hooks'
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
 * WantedCutoffUnmet lists monitored episodes with a file that are still
 * below the quality profile's cutoff — i.e. upgrade candidates. Backed by
 * GET /api/v3/wanted/cutoff (paged).
 *
 * Note: the backend currently returns "any monitored episode with a file"
 * as an operational approximation; the real cutoff rule is enforced by
 * the decision engine when ranking grab candidates.
 */
export function WantedCutoffUnmet() {
  const [page, setPage] = useState(1)
  const pageSize = 50
  const { data, isLoading, isError, error } = useWantedCutoffUnmet(page, pageSize)

  const records: WantedEpisodeV3[] = data?.records ?? []
  const total = data?.totalRecords ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Cutoff Unmet</h1>
      <p className={styles.subtitle}>
        Episodes with a file that may still be upgraded when a release above
        the profile&apos;s cutoff appears. The decision engine handles the
        actual upgrade choice; this page is informational.
      </p>

      {isLoading && <p className={styles.status}>Loading upgrade candidates…</p>}
      {isError && (
        <p className={styles.error}>
          Failed to load: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}

      {!isLoading && records.length === 0 && !isError && (
        <p className={styles.status}>
          No upgrade candidates. Every monitored episode with a file already
          meets its profile cutoff.
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
                      <span className={styles.pillUpgrade}>Has file</span>
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
