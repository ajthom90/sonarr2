import { NavLink } from 'react-router-dom'
import styles from './Sidebar.module.css'

const navItems = [
  { to: '/', label: 'Series', icon: '📺' },
  { to: '/calendar', label: 'Calendar', icon: '📅' },
  { to: '/activity', label: 'Activity', icon: '⚡' },
  { to: '/wanted', label: 'Wanted', icon: '🔍' },
  { to: '/settings', label: 'Settings', icon: '⚙️' },
  { to: '/system', label: 'System', icon: '💻' },
]

export function Sidebar() {
  return (
    <nav className={styles.sidebar}>
      <div className={styles.logo}>sonarr2</div>
      <ul>
        {navItems.map(item => (
          <li key={item.to}>
            <NavLink to={item.to} className={({ isActive }) => isActive ? styles.active : ''}>
              <span>{item.icon}</span> {item.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}
