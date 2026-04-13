import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Modal } from './Modal'
import { useDeleteSeries } from '../api/hooks'
import styles from './DeleteSeriesModal.module.css'

interface DeleteSeriesModalProps {
  seriesId: number
  title: string
  open: boolean
  onClose: () => void
}

export function DeleteSeriesModal({ seriesId, title, open, onClose }: DeleteSeriesModalProps) {
  const deleteSeries = useDeleteSeries()
  const navigate = useNavigate()
  const [deleteFiles, setDeleteFiles] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function handleDelete() {
    setError(null)
    deleteSeries.mutate(
      { id: seriesId, deleteFiles },
      {
        onSuccess: () => {
          onClose()
          navigate('/')
        },
        onError: (err) => {
          setError(err instanceof Error ? err.message : 'Failed to delete series')
        },
      },
    )
  }

  return (
    <Modal open={open} onClose={onClose} title="Delete Series">
      <p className={styles.message}>
        Are you sure you want to delete <strong>{title}</strong>?
      </p>

      <label className={styles.checkboxLabel}>
        <input
          type="checkbox"
          checked={deleteFiles}
          onChange={(e) => setDeleteFiles(e.target.checked)}
          className={styles.checkbox}
        />
        Delete files from disk
      </label>

      {error && <p className={styles.error}>{error}</p>}

      <div className={styles.actions}>
        <button
          className={styles.deleteBtn}
          onClick={handleDelete}
          disabled={deleteSeries.isPending}
        >
          {deleteSeries.isPending ? 'Deleting...' : 'Delete'}
        </button>
        <button className={styles.cancelBtn} onClick={onClose}>
          Cancel
        </button>
      </div>
    </Modal>
  )
}
