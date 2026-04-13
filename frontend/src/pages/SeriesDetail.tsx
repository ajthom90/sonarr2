import { useParams, Link } from 'react-router-dom'
import { useSeries, useEpisodes } from '../api/hooks'
import type { Episode } from '../api/types'
import styles from './SeriesDetail.module.css'

function StatusBadge({ status }: { status: string }) {
  const cls =
    status.toLowerCase() === 'continuing'
      ? styles.badgeContinuing
      : status.toLowerCase() === 'ended'
        ? styles.badgeEnded
        : status.toLowerCase() === 'upcoming'
          ? styles.badgeUpcoming
          : styles.badgeDefault
  return <span className={`${styles.badge} ${cls ?? ''}`}>{status}</span>
}

function formatAirDate(dateStr?: string): string {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function groupBySeason(episodes: Episode[]): Map<number, Episode[]> {
  const map = new Map<number, Episode[]>()
  for (const ep of episodes) {
    const bucket = map.get(ep.seasonNumber) ?? []
    bucket.push(ep)
    map.set(ep.seasonNumber, bucket)
  }
  // Sort episodes within each season
  for (const [, eps] of map) {
    eps.sort((a, b) => a.episodeNumber - b.episodeNumber)
  }
  return map
}

export function SeriesDetail() {
  const { id } = useParams<{ id: string }>()
  const seriesId = Number(id)

  const { data: series, isLoading: seriesLoading, isError: seriesError, error: seriesErr } = useSeries(seriesId)
  const { data: episodesPage, isLoading: episodesLoading } = useEpisodes(seriesId)

  if (seriesLoading) {
    return (
      <div className={styles.page}>
        <p className={styles.stateMessage}>Loading...</p>
      </div>
    )
  }

  if (seriesError || !series) {
    return (
      <div className={styles.page}>
        <p className={styles.errorMessage}>
          Failed to load series: {seriesErr instanceof Error ? seriesErr.message : 'Unknown error'}
        </p>
        <Link to="/series" className={styles.backLink}>← Back to Series</Link>
      </div>
    )
  }

  const episodes = episodesPage?.data ?? []
  const seasonMap = groupBySeason(episodes)
  const seasons = [...seasonMap.keys()].sort((a, b) => a - b)

  return (
    <div className={styles.page}>
      {/* Header */}
      <div className={styles.header}>
        <Link to="/series" className={styles.backLink}>← Series</Link>
        <div className={styles.titleRow}>
          <h1 className={styles.title}>{series.title}</h1>
          {series.year > 0 && <span className={styles.year}>({series.year})</span>}
          <StatusBadge status={series.status} />
        </div>
        <dl className={styles.meta}>
          <div className={styles.metaItem}>
            <dt className={styles.metaLabel}>Type</dt>
            <dd className={styles.metaValue}>{series.seriesType || '—'}</dd>
          </div>
          <div className={styles.metaItem}>
            <dt className={styles.metaLabel}>Path</dt>
            <dd className={`${styles.metaValue} ${styles.metaPath}`}>{series.path || '—'}</dd>
          </div>
          {series.statistics && (
            <>
              <div className={styles.metaItem}>
                <dt className={styles.metaLabel}>Episodes</dt>
                <dd className={styles.metaValue}>
                  {series.statistics.episodeFileCount}/{series.statistics.episodeCount}
                </dd>
              </div>
              <div className={styles.metaItem}>
                <dt className={styles.metaLabel}>Size</dt>
                <dd className={styles.metaValue}>
                  {(series.statistics.sizeOnDisk / 1e9).toFixed(1)} GB
                </dd>
              </div>
            </>
          )}
        </dl>
      </div>

      {/* Episodes */}
      <div className={styles.episodesSection}>
        <h2 className={styles.sectionHeading}>Episodes</h2>

        {episodesLoading && <p className={styles.stateMessage}>Loading episodes...</p>}

        {!episodesLoading && episodes.length === 0 && (
          <p className={styles.stateMessage}>No episodes found.</p>
        )}

        {seasons.map((seasonNum) => {
          const eps = seasonMap.get(seasonNum) ?? []
          const monitoredCount = eps.filter((e) => e.monitored).length
          const hasFileCount = eps.filter((e) => e.hasFile).length

          return (
            <div key={seasonNum} className={styles.season}>
              <div className={styles.seasonHeader}>
                <span className={styles.seasonTitle}>
                  {seasonNum === 0 ? 'Specials' : `Season ${seasonNum}`}
                </span>
                <span className={styles.seasonStats}>
                  {hasFileCount}/{eps.length} episodes &middot; {monitoredCount} monitored
                </span>
              </div>

              <div className={styles.tableWrapper}>
                <table className={styles.table}>
                  <thead>
                    <tr>
                      <th className={styles.th}>#</th>
                      <th className={styles.th}>Title</th>
                      <th className={styles.th}>Air Date</th>
                      <th className={styles.th}>File</th>
                      <th className={styles.th}>Monitored</th>
                    </tr>
                  </thead>
                  <tbody>
                    {eps.map((ep) => (
                      <tr key={ep.id} className={styles.row}>
                        <td className={`${styles.td} ${styles.epNum}`}>
                          {ep.episodeNumber}
                        </td>
                        <td className={styles.td}>
                          <span className={styles.epTitle}>{ep.title || 'TBA'}</span>
                        </td>
                        <td className={`${styles.td} ${styles.airDate}`}>
                          {formatAirDate(ep.airDate)}
                        </td>
                        <td className={styles.td}>
                          {ep.hasFile ? (
                            <span className={styles.hasFile} title="File exists">&#10003;</span>
                          ) : (
                            <span className={styles.missingFile} title="No file">&#10007;</span>
                          )}
                        </td>
                        <td className={styles.td}>
                          <span
                            className={ep.monitored ? styles.monitoredDotOn : styles.monitoredDotOff}
                            title={ep.monitored ? 'Monitored' : 'Unmonitored'}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
