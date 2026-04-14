import type { ReactNode } from 'react'
import styles from './ProviderListSection.module.css'

export interface ProviderListSectionProps<T> {
  title: string
  items: T[]
  emptyMessage: string
  renderCard: (item: T) => ReactNode
  onAdd: () => void
}

export function ProviderListSection<T>({
  title,
  items,
  emptyMessage,
  renderCard,
  onAdd,
}: ProviderListSectionProps<T>) {
  return (
    <section className={styles.section}>
      <div className={styles.header}>
        <h2 className={styles.title}>{title}</h2>
        <button type="button" className={styles.addBtn} onClick={onAdd}>
          + Add
        </button>
      </div>
      {items.length === 0 ? (
        <p className={styles.empty}>{emptyMessage}</p>
      ) : (
        <div className={styles.grid}>
          {items.map((item, i) => (
            <div
              key={(item as { id?: number }).id ?? i}
              className={styles.cardWrap}
            >
              {renderCard(item)}
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
