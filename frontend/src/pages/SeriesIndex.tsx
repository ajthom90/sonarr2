import { Link } from 'react-router-dom'
import { useSeriesList } from '../api/hooks'
import type { Series } from '../api/types'
import styles from './SeriesIndex.module.css'

function statusBadgeClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'continuing':
      return styles.badgeContinuing ?? ''
    case 'ended':
      return styles.badgeEnded ?? ''
    case 'upcoming':
      return styles.badgeUpcoming ?? ''
    default:
      return styles.badgeDefault ?? ''
  }
}

function formatSize(bytes: number): string {
  return (bytes / 1e9).toFixed(1) + ' GB'
}

function EpisodeProgress({ series }: { series: Series }) {
  const stats = series.statistics
  if (!stats) return <span className={styles.muted}>—</span>
  const { episodeFileCount, episodeCount } = stats
  const pct = episodeCount > 0 ? Math.round((episodeFileCount / episodeCount) * 100) : 0
  return (
    <span>
      {episodeFileCount}/{episodeCount}
      <span className={styles.muted}> ({pct}%)</span>
    </span>
  )
}

export function SeriesIndex() {
  const { data, isLoading, isError, error } = useSeriesList()

  if (isLoading) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Series</h1>
        <p className={styles.stateMessage}>Loading...</p>
      </div>
    )
  }

  if (isError) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Series</h1>
        <p className={styles.errorMessage}>
          Failed to load series: {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      </div>
    )
  }

  const series = data?.data ?? []

  if (series.length === 0) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Series</h1>
        <p className={styles.stateMessage}>No series added yet. Add a series to get started.</p>
      </div>
    )
  }

  const sorted = [...series].sort((a, b) => a.sortTitle.localeCompare(b.sortTitle))

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Series</h1>
      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.th}>Title</th>
              <th className={styles.th}>Status</th>
              <th className={styles.th}>Episodes</th>
              <th className={styles.th}>Size</th>
              <th className={styles.th}>Monitored</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((s) => (
              <tr key={s.id} className={styles.row}>
                <td className={styles.td}>
                  <Link to={`/series/${s.id}`} className={styles.titleLink}>
                    {s.title}
                  </Link>
                </td>
                <td className={styles.td}>
                  <span className={`${styles.badge} ${statusBadgeClass(s.status)}`}>
                    {s.status}
                  </span>
                </td>
                <td className={styles.td}>
                  <EpisodeProgress series={s} />
                </td>
                <td className={styles.td}>
                  {s.statistics ? formatSize(s.statistics.sizeOnDisk) : <span className={styles.muted}>—</span>}
                </td>
                <td className={styles.td}>
                  <span
                    className={s.monitored ? styles.monitoredDotOn : styles.monitoredDotOff}
                    title={s.monitored ? 'Monitored' : 'Unmonitored'}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
