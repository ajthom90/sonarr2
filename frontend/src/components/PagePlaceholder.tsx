import type { ReactNode } from 'react'
import styles from './PagePlaceholder.module.css'

type Props = {
  title: string
  description?: string
  children?: ReactNode
}

/**
 * PagePlaceholder renders a page-level scaffolding used for routes whose
 * backend feature is either in-flight or not yet implemented. It ensures
 * every sidebar menu item resolves to a visible page (no 404s for
 * scaffolded routes), matching Sonarr's navigation shape exactly.
 */
export function PagePlaceholder({ title, description, children }: Props) {
  return (
    <div className={styles.page}>
      <h1 className={styles.title}>{title}</h1>
      {description && <p className={styles.description}>{description}</p>}
      {children}
    </div>
  )
}
