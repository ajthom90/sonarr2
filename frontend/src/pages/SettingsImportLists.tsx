import { useImportListSchema } from '../api/hooks'
import styles from './ProviderCatalog.module.css'

/**
 * SettingsImportLists is a browsable catalog of registered Import List
 * providers. The backend currently exposes GET /api/v3/importlist/schema
 * (providers) + GET /api/v3/importlist (always [] — no store yet). CRUD is
 * a follow-up sub-project.
 */
export function SettingsImportLists() {
  const { data: schemas = [], isLoading, isError, error } = useImportListSchema()

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Import Lists</h1>
      <div className={styles.banner}>
        <strong>Preview.</strong> The {schemas.length || 'registered'} providers
        below are wired on the server but can&apos;t be saved yet — persisted
        Import List instances arrive in a follow-up sub-project. For now this
        page shows you what sonarr2 supports.
      </div>

      {isLoading && <p className={styles.status}>Loading providers…</p>}
      {isError && (
        <p className={styles.error}>
          Failed to load: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}

      {!isLoading && schemas.length === 0 && !isError && (
        <p className={styles.status}>No providers registered.</p>
      )}

      {schemas.length > 0 && (
        <div className={styles.grid}>
          {schemas.map((p) => (
            <div key={p.implementation} className={styles.card}>
              <div className={styles.cardHeader}>
                <div className={styles.cardName}>{p.name}</div>
                <div className={styles.cardImpl}>{p.implementation}</div>
              </div>
              <div className={styles.cardFooter}>
                <span className={styles.notYet}>Not yet configurable</span>
                <span className={styles.fieldCount}>
                  {p.fields.length} settings field{p.fields.length === 1 ? '' : 's'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
