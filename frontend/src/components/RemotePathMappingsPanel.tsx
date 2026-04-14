import { useState } from 'react'
import {
  useRemotePathMappings,
  useCreateRemotePathMapping,
  useDeleteRemotePathMapping,
} from '../api/hooks'
import styles from './RemotePathMappingsPanel.module.css'

export function RemotePathMappingsPanel() {
  const { data: mappings = [] } = useRemotePathMappings()
  const create = useCreateRemotePathMapping()
  const del = useDeleteRemotePathMapping()

  const [host, setHost] = useState('')
  const [remotePath, setRemotePath] = useState('')
  const [localPath, setLocalPath] = useState('')

  async function handleAdd() {
    if (!host || !remotePath || !localPath) return
    try {
      await create.mutateAsync({ host, remotePath, localPath })
      setHost('')
      setRemotePath('')
      setLocalPath('')
    } catch (err) {
      alert((err as Error).message)
    }
  }

  async function handleDelete(id: number) {
    if (!confirm('Remove this mapping?')) return
    try {
      await del.mutateAsync(id)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.title}>Remote Path Mappings</h2>
      <p className={styles.help}>
        Translate paths from a download client&apos;s host to paths visible on this server.
      </p>

      {mappings.length > 0 ? (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Host</th>
              <th>Remote Path</th>
              <th>Local Path</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {mappings.map((m) => (
              <tr key={m.id}>
                <td>{m.host}</td>
                <td>
                  <code>{m.remotePath}</code>
                </td>
                <td>
                  <code>{m.localPath}</code>
                </td>
                <td>
                  <button
                    className={styles.deleteBtn}
                    onClick={() => handleDelete(m.id)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className={styles.empty}>No mappings yet.</p>
      )}

      <div className={styles.addRow}>
        <input
          placeholder="download-client.host"
          value={host}
          onChange={(e) => setHost(e.target.value)}
        />
        <input
          placeholder="/remote/path"
          value={remotePath}
          onChange={(e) => setRemotePath(e.target.value)}
        />
        <input
          placeholder="/local/path"
          value={localPath}
          onChange={(e) => setLocalPath(e.target.value)}
        />
        <button className={styles.addBtn} onClick={handleAdd}>
          Add
        </button>
      </div>
    </section>
  )
}
