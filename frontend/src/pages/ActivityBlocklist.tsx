import { useEffect, useState } from 'react'
import { apiFetchRaw } from '../api/client'
import { PagePlaceholder } from '../components/PagePlaceholder'

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

/**
 * ActivityBlocklist renders the /activity/blocklist page, showing releases
 * that have been manually blocklisted or auto-blocklisted after failed import.
 * Wires to /api/v3/blocklist. Selecting entries allows bulk removal.
 */
export function ActivityBlocklist() {
  const [page, setPage] = useState<Paged | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const res = await apiFetchRaw('/api/v3/blocklist?page=1&pageSize=50')
      const body = (await res.json()) as Paged
      setPage(body)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  async function removeOne(id: number) {
    await apiFetchRaw(`/api/v3/blocklist/${id}`, { method: 'DELETE' })
    load()
  }

  async function removeBulk() {
    if (selected.size === 0) return
    await apiFetchRaw('/api/v3/blocklist/bulk', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids: Array.from(selected) }),
    })
    setSelected(new Set())
    load()
  }

  function toggle(id: number) {
    setSelected(s => {
      const n = new Set(s)
      if (n.has(id)) n.delete(id); else n.add(id)
      return n
    })
  }

  if (loading) return <PagePlaceholder title="Blocklist" description="Loading..." />
  if (error) return <PagePlaceholder title="Blocklist" description={`Error: ${error}`} />
  const entries = page?.records ?? []

  return (
    <PagePlaceholder
      title="Blocklist"
      description="Releases that should not be grabbed again. Rows are added manually or when an import fails and auto-blocklist is enabled."
    >
      {selected.size > 0 && (
        <button onClick={removeBulk} style={{ marginBottom: 12 }}>
          Remove {selected.size} selected
        </button>
      )}
      {entries.length === 0 ? (
        <p>No entries.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: 8 }}></th>
              <th style={{ textAlign: 'left', padding: 8 }}>Release</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Indexer</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Protocol</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Date</th>
              <th style={{ textAlign: 'left', padding: 8 }}></th>
            </tr>
          </thead>
          <tbody>
            {entries.map(e => (
              <tr key={e.id}>
                <td style={{ padding: 8 }}>
                  <input type="checkbox" checked={selected.has(e.id)} onChange={() => toggle(e.id)} />
                </td>
                <td style={{ padding: 8 }}>{e.sourceTitle}</td>
                <td style={{ padding: 8 }}>{e.indexer}</td>
                <td style={{ padding: 8 }}>{e.protocol}</td>
                <td style={{ padding: 8 }}>{new Date(e.date).toLocaleString()}</td>
                <td style={{ padding: 8 }}>
                  <button onClick={() => removeOne(e.id)}>Remove</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PagePlaceholder>
  )
}
