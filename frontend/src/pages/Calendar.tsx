import { useState } from 'react'
import { useCalendar } from '../api/hooks'
import type { Episode } from '../api/types'
import styles from './Calendar.module.css'

/** Return Monday of the week containing `date`. */
function getWeekStart(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay() // 0=Sun … 6=Sat
  const diff = (day === 0 ? -6 : 1 - day) // shift to Monday
  d.setDate(d.getDate() + diff)
  d.setHours(0, 0, 0, 0)
  return d
}

/** Add `n` days to a Date (returns new Date). */
function addDays(date: Date, n: number): Date {
  const d = new Date(date)
  d.setDate(d.getDate() + n)
  return d
}

/** Format a Date as `YYYY-MM-DD`. */
function toDateStr(date: Date): string {
  return date.toISOString().slice(0, 10)
}

/** Format a Date as a human-readable day label, e.g. "Mon Apr 10". */
function dayLabel(date: Date): string {
  return date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
}

/** Format the week range for display. */
function weekRangeLabel(start: Date, end: Date): string {
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
  const startStr = start.toLocaleDateString('en-US', opts)
  const endStr = end.toLocaleDateString('en-US', { ...opts, year: 'numeric' })
  return `${startStr} – ${endStr}`
}

/** Parse the air date string and return a `YYYY-MM-DD` key. */
function airDateKey(ep: Episode): string | null {
  const raw = ep.airDate ?? ep.airDateUtc
  if (!raw) return null
  return raw.slice(0, 10)
}

/** Format an air-time from an ISO UTC string, shown in local time. */
function formatAirTime(utc?: string): string {
  if (!utc) return ''
  try {
    return new Date(utc).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })
  } catch {
    return ''
  }
}

/** Format SxxExx from season + episode numbers. */
function formatEpCode(season: number, episode: number): string {
  return `S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
}

export function Calendar() {
  const [weekStart, setWeekStart] = useState<Date>(() => getWeekStart(new Date()))
  const weekEnd = addDays(weekStart, 6)

  const startStr = toDateStr(weekStart)
  const endStr = toDateStr(weekEnd)

  const { data, isLoading, isError, error } = useCalendar(startStr, endStr)

  const episodes: Episode[] = data?.data ?? []

  // Group episodes by their local air date
  const grouped = new Map<string, Episode[]>()
  for (let i = 0; i < 7; i++) {
    const key = toDateStr(addDays(weekStart, i))
    grouped.set(key, [])
  }
  for (const ep of episodes) {
    const key = airDateKey(ep)
    if (key && grouped.has(key)) {
      grouped.get(key)!.push(ep)
    }
  }

  function prevWeek() {
    setWeekStart((w) => addDays(w, -7))
  }
  function nextWeek() {
    setWeekStart((w) => addDays(w, 7))
  }

  const totalEpisodes = episodes.length

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Calendar</h1>

      <div className={styles.nav}>
        <button className={styles.navBtn} onClick={prevWeek}>
          {'← Prev'}
        </button>
        <span className={styles.weekLabel}>{weekRangeLabel(weekStart, weekEnd)}</span>
        <button className={styles.navBtn} onClick={nextWeek}>
          {'Next →'}
        </button>
      </div>

      {isLoading && <p className={styles.stateMessage}>Loading episodes...</p>}

      {isError && (
        <p className={styles.errorMessage}>
          Failed to load calendar:{' '}
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      )}

      {!isLoading && !isError && totalEpisodes === 0 && (
        <p className={styles.stateMessage}>No episodes airing this week.</p>
      )}

      {!isLoading && !isError && totalEpisodes > 0 && (
        <>
          {Array.from(grouped.entries()).map(([dateKey, eps]) => {
            if (eps.length === 0) return null
            const dateObj = new Date(dateKey + 'T00:00:00')
            return (
              <div key={dateKey} className={styles.dayGroup}>
                <div className={styles.dayHeader}>{dayLabel(dateObj)}</div>
                <ul className={styles.episodeList}>
                  {eps.map((ep) => (
                    <li key={ep.id} className={styles.episodeItem}>
                      <span className={styles.airTime}>
                        {formatAirTime(ep.airDateUtc)}
                      </span>
                      <span className={styles.episodeNum}>
                        {formatEpCode(ep.seasonNumber, ep.episodeNumber)}
                      </span>
                      <span className={styles.episodeTitle}>
                        {ep.title || 'TBA'}
                      </span>
                      <span className={styles.seriesId}>Series #{ep.seriesId}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )
          })}
        </>
      )}
    </div>
  )
}
