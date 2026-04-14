import { useState, useRef, useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useConnectionStatus, useSeriesList } from '../api/hooks'
import type { Series } from '../api/types'
import styles from './Sidebar.module.css'

const navItems = [
  {
    to: '/', label: 'Series', icon: '📺', children: [
      { to: '/add/new', label: 'Add New' },
      { to: '/add/import', label: 'Library Import' },
      { to: '/serieseditor', label: 'Mass Editor' },
      { to: '/seasonpass', label: 'Season Pass' },
    ],
  },
  { to: '/calendar', label: 'Calendar', icon: '📅' },
  {
    to: '/activity', label: 'Activity', icon: '⚡', children: [
      { to: '/activity/queue', label: 'Queue' },
      { to: '/activity/history', label: 'History' },
      { to: '/activity/blocklist', label: 'Blocklist' },
    ],
  },
  {
    to: '/wanted', label: 'Wanted', icon: '🔍', children: [
      { to: '/wanted/missing', label: 'Missing' },
      { to: '/wanted/cutoffunmet', label: 'Cutoff Unmet' },
    ],
  },
  {
    to: '/settings', label: 'Settings', icon: '⚙️', children: [
      { to: '/settings/mediamanagement', label: 'Media Management' },
      { to: '/settings/profiles', label: 'Profiles' },
      { to: '/settings/quality', label: 'Quality' },
      { to: '/settings/customformats', label: 'Custom Formats' },
      { to: '/settings/indexers', label: 'Indexers' },
      { to: '/settings/downloadclients', label: 'Download Clients' },
      { to: '/settings/importlists', label: 'Import Lists' },
      { to: '/settings/connect', label: 'Connect' },
      { to: '/settings/metadata', label: 'Metadata' },
      { to: '/settings/metadatasource', label: 'Metadata Source' },
      { to: '/settings/tags', label: 'Tags' },
      { to: '/settings/general', label: 'General' },
      { to: '/settings/ui', label: 'UI' },
    ],
  },
  {
    to: '/system', label: 'System', icon: '💻', children: [
      { to: '/system/status', label: 'Status' },
      { to: '/system/tasks', label: 'Tasks' },
      { to: '/system/backup', label: 'Backup' },
      { to: '/system/updates', label: 'Updates' },
      { to: '/system/events', label: 'Events' },
      { to: '/system/logs/files', label: 'Log Files' },
    ],
  },
]

function SearchBar() {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const { data: seriesPage } = useSeriesList()
  const navigate = useNavigate()
  const wrapperRef = useRef<HTMLDivElement>(null)

  const allSeries: Series[] = seriesPage?.data ?? []
  const filtered = query.length >= 1
    ? allSeries.filter((s) => s.title.toLowerCase().includes(query.toLowerCase())).slice(0, 10)
    : []

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  function handleSelect(series: Series) {
    setQuery('')
    setOpen(false)
    navigate(`/series/${series.id}`)
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Escape') {
      setOpen(false)
      setQuery('')
    }
  }

  return (
    <div className={styles.searchWrapper} ref={wrapperRef}>
      <input
        className={styles.searchInput}
        type="text"
        placeholder="Search series..."
        value={query}
        onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
        onFocus={() => { if (query.length >= 1) setOpen(true) }}
        onKeyDown={handleKeyDown}
      />
      {open && filtered.length > 0 && (
        <ul className={styles.searchResults}>
          {filtered.map((s) => (
            <li key={s.id}>
              <button
                className={styles.searchResultItem}
                onClick={() => handleSelect(s)}
              >
                <span className={styles.searchResultTitle}>{s.title}</span>
                {s.year > 0 && <span className={styles.searchResultYear}>({s.year})</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
      {open && query.length >= 1 && filtered.length === 0 && (
        <div className={styles.searchNoResults}>No series found</div>
      )}
    </div>
  )
}

export function Sidebar() {
  const { data: connected, isLoading } = useConnectionStatus()

  const dotClass = isLoading
    ? styles.dotLoading
    : connected
      ? styles.dotConnected
      : styles.dotDisconnected

  const dotTitle = isLoading
    ? 'Checking connection...'
    : connected
      ? 'Connected'
      : 'Disconnected'

  return (
    <nav className={styles.sidebar}>
      <div className={styles.logo}>
        sonarr2
        <span className={`${styles.statusDot} ${dotClass}`} title={dotTitle} />
      </div>
      <SearchBar />
      <ul>
        {navItems.map(item => (
          <li key={item.to}>
            <NavLink to={item.to} end={!!item.children} className={({ isActive }) => isActive ? styles.active : ''}>
              <span>{item.icon}</span> {item.label}
            </NavLink>
            {item.children && (
              <ul className={styles.subNav}>
                {item.children.map(child => (
                  <li key={child.to}>
                    <NavLink to={child.to} className={({ isActive }) => isActive ? styles.active : ''}>
                      {child.label}
                    </NavLink>
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ul>
    </nav>
  )
}
