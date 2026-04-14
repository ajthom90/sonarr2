import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetchRaw } from '../api/client'
import { apiV3 } from '../api/v3'
import styles from './ActivityBlocklist.module.css'

type BlocklistEntry = {
  id: number
  seriesId: number
  episodeIds: number[]
  sourceTitle: string
  date: string
  protocol: string
  indexer: string
  message: string
}

type Paged = {
  page: number
  pageSize: number
  totalRecords: number
  records: BlocklistEntry[]
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

/**
 * ActivityBlocklist renders /activity/blocklist — releases that should not
 * be grabbed again. Users can select rows and remove them in bulk.
 * Wires to /api/v3/blocklist via apiV3.
 */
export function ActivityBlocklist() {
  const qc = useQueryClient()
  const [selected, setSelected] = useState<Set<number>>(new Set())

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['v3', 'blocklist', 1, 50],
    queryFn: () => apiV3.get<Paged>('/blocklist?page=1&pageSize=50'),
  })

  const removeOne = useMutation({
    mutationFn: (id: number) => apiV3.delete(`/blocklist/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'blocklist'] }),
  })

  // Bulk-delete needs a body in the DELETE request, which our apiV3.delete
  // helper doesn't support yet (kept narrow deliberately). Small escape
  // hatch via apiFetchRaw is cleaner than widening the helper for one
  // endpoint. TODO: extend apiV3.delete to accept an optional body so this
  // pattern consolidates.
  const [bulkBusy, setBulkBusy] = useState(false)
  async function handleBulkRemove() {
    if (selected.size === 0) return
    setBulkBusy(true)
    try {
      const res = await apiFetchRaw('/api/v3/blocklist/bulk', {
        method: 'DELETE',
        body: JSON.stringify({ ids: Array.from(selected) }),
      })
      if (!res.ok) {
        const body = (await res.json().catch(() => ({}))) as { message?: string }
        alert(`Failed to remove: ${body.message ?? res.statusText}`)
        return
      }
      setSelected(new Set())
      qc.invalidateQueries({ queryKey: ['v3', 'blocklist'] })
    } finally {
      setBulkBusy(false)
    }
  }

  function toggle(id: number) {
    setSelected((s) => {
      const n = new Set(s)
      if (n.has(id)) n.delete(id)
      else n.add(id)
      return n
    })
  }

  const entries = data?.records ?? []

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Blocklist</h1>
      <p className={styles.subtitle}>
        Releases that should not be grabbed again. Rows are added manually or
        when an import fails and auto-blocklist is enabled.
      </p>

      {isLoading && <p className={styles.status}>Loading blocklist…</p>}
      {isError && (
        <p className={styles.error}>
          Failed to load: {error instanceof Error ? error.message : 'unknown error'}
        </p>
      )}

      {!isLoading && entries.length === 0 && !isError && (
        <p className={styles.status}>No entries.</p>
      )}

      {entries.length > 0 && (
        <>
          <div className={styles.toolbar}>
            <button
              className={styles.removeBulkBtn}
              disabled={selected.size === 0 || bulkBusy}
              onClick={handleBulkRemove}
            >
              Remove {selected.size} selected
            </button>
            <span className={styles.pagerInfo}>
              {data?.totalRecords ?? entries.length} entr
              {(data?.totalRecords ?? entries.length) === 1 ? 'y' : 'ies'}
            </span>
          </div>

          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th className={styles.th}></th>
                  <th className={styles.th}>Release</th>
                  <th className={styles.th}>Indexer</th>
                  <th className={styles.th}>Protocol</th>
                  <th className={styles.th}>Date</th>
                  <th className={styles.th}></th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id} className={styles.row}>
                    <td className={styles.td}>
                      <input
                        type="checkbox"
                        checked={selected.has(e.id)}
                        onChange={() => toggle(e.id)}
                      />
                    </td>
                    <td className={styles.td}>
                      <span className={styles.sourceTitle} title={e.sourceTitle}>
                        {e.sourceTitle}
                      </span>
                    </td>
                    <td className={styles.td}>{e.indexer || '—'}</td>
                    <td className={styles.td}>{e.protocol || '—'}</td>
                    <td className={styles.td}>{formatDate(e.date)}</td>
                    <td className={styles.td}>
                      <button
                        className={styles.removeBtn}
                        onClick={() => removeOne.mutate(e.id)}
                        disabled={removeOne.isPending}
                      >
                        Remove
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}
