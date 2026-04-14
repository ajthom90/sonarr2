import { useMetadataSchema } from '../api/hooks'
import styles from './ProviderCatalog.module.css'

/**
 * SettingsMetadata is a browsable catalog of registered Metadata consumers
 * (file-emitting writers for Kodi/XBMC, Plex, Roksbox, WDTV). The backend
 * currently exposes GET /api/v3/metadata/schema and GET /api/v3/metadata
 * (always [] — no instance store). CRUD is a follow-up sub-project.
 *
 * Separate from /settings/metadatasource — that page configures the source
 * catalog (TheTVDB); this page configures file output for media players.
 */
export function SettingsMetadata() {
  const { data: schemas = [], isLoading, isError, error } = useMetadataSchema()

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Metadata</h1>
      <div className={styles.banner}>
        <strong>Preview.</strong> Metadata consumers emit .nfo + image files
        into series folders so media players like Kodi, Plex, Emby, and WDTV
        can read them. The {schemas.length || 'registered'} consumers below
        are wired on the server but can&apos;t be saved yet — persisted
        configuration is a follow-up sub-project.
      </div>

      {isLoading && <p className={styles.status}>Loading consumers…</p>}
      {isError && (
        <p className={styles.error}>
          Failed to load: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}

      {!isLoading && schemas.length === 0 && !isError && (
        <p className={styles.status}>No consumers registered.</p>
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
                  {p.fields.length} setting{p.fields.length === 1 ? '' : 's'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
