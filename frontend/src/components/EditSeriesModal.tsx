import { useState, useEffect } from 'react'
import { Modal } from './Modal'
import { useUpdateSeries, useQualityProfiles } from '../api/hooks'
import type { Series, QualityProfile } from '../api/types'
import styles from './EditSeriesModal.module.css'

interface EditSeriesModalProps {
  series: Series
  open: boolean
  onClose: () => void
}

const SERIES_TYPES = ['standard', 'daily', 'anime']

export function EditSeriesModal({ series, open, onClose }: EditSeriesModalProps) {
  const updateSeries = useUpdateSeries()
  const { data: profilesPage } = useQualityProfiles()
  const profiles: QualityProfile[] = profilesPage?.data ?? []

  const [monitored, setMonitored] = useState(series.monitored)
  const [seriesType, setSeriesType] = useState(series.seriesType)
  const [path, setPath] = useState(series.path)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setMonitored(series.monitored)
      setSeriesType(series.seriesType)
      setPath(series.path)
      setError(null)
    }
  }, [open, series])

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    updateSeries.mutate(
      {
        id: series.id,
        monitored,
        seriesType,
        path,
      },
      {
        onSuccess: () => onClose(),
        onError: (err) => {
          setError(err instanceof Error ? err.message : 'Failed to update series')
        },
      },
    )
  }

  return (
    <Modal open={open} onClose={onClose} title={`Edit ${series.title}`}>
      <form onSubmit={handleSave}>
        <div className={styles.field}>
          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={monitored}
              onChange={(e) => setMonitored(e.target.checked)}
              className={styles.checkbox}
            />
            Monitored
          </label>
        </div>

        {profiles.length > 0 && (
          <div className={styles.field}>
            <label className={styles.label}>Quality Profile</label>
            <select className={styles.select} disabled>
              {profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
            <span className={styles.hint}>Profile changes are not yet supported</span>
          </div>
        )}

        <div className={styles.field}>
          <label className={styles.label}>Series Type</label>
          <select
            className={styles.select}
            value={seriesType}
            onChange={(e) => setSeriesType(e.target.value)}
          >
            {SERIES_TYPES.map((t) => (
              <option key={t} value={t}>
                {t.charAt(0).toUpperCase() + t.slice(1)}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.field}>
          <label className={styles.label}>Path</label>
          <input
            type="text"
            className={styles.input}
            value={path}
            onChange={(e) => setPath(e.target.value)}
          />
        </div>

        {error && <p className={styles.error}>{error}</p>}

        <div className={styles.actions}>
          <button
            type="submit"
            className={styles.saveBtn}
            disabled={updateSeries.isPending}
          >
            {updateSeries.isPending ? 'Saving...' : 'Save'}
          </button>
          <button type="button" className={styles.cancelBtn} onClick={onClose}>
            Cancel
          </button>
        </div>
      </form>
    </Modal>
  )
}
