import { NavLink } from 'react-router-dom'
import { useConnectionStatus } from '../api/hooks'
import styles from './Sidebar.module.css'

const navItems = [
  { to: '/', label: 'Series', icon: '📺' },
  { to: '/add/new', label: 'Add New', icon: '➕' },
  { to: '/calendar', label: 'Calendar', icon: '📅' },
  { to: '/activity', label: 'Activity', icon: '⚡' },
  { to: '/wanted', label: 'Wanted', icon: '🔍' },
  { to: '/settings', label: 'Settings', icon: '⚙️', children: [
    { to: '/settings/general', label: 'General' },
    { to: '/settings/mediamanagement', label: 'Media Management' },
    { to: '/settings/profiles', label: 'Profiles' },
    { to: '/settings/customformats', label: 'Custom Formats' },
  ] },
  { to: '/system', label: 'System', icon: '💻' },
]

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
