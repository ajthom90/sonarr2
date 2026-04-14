import { useEffect, useState } from 'react'
import { apiFetchRaw } from '../api/client'
import { PagePlaceholder } from '../components/PagePlaceholder'

type Tag = { id: number; label: string }
type TagDetails = Tag & {
  delayProfileIds: number[]
  importListIds: number[]
  notificationIds: number[]
  restrictionIds: number[]
  indexerIds: number[]
  downloadClientIds: number[]
  autoTagIds: number[]
  seriesIds: number[]
}

/**
 * SettingsTags renders /settings/tags. Tags are labels attached to series,
 * indexers, download clients, notifications, and profiles, driving behavior
 * like per-tag delay profiles and release profiles. Sonarr's shape.
 */
export function SettingsTags() {
  const [tags, setTags] = useState<TagDetails[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [newLabel, setNewLabel] = useState('')

  async function load() {
    setLoading(true); setError(null)
    try {
      const res = await apiFetchRaw('/api/v3/tag/detail')
      setTags(await res.json())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { load() }, [])

  async function add() {
    if (!newLabel.trim()) return
    await apiFetchRaw('/api/v3/tag', {
      method: 'POST',
      body: JSON.stringify({ label: newLabel.trim() }),
    })
    setNewLabel('')
    load()
  }

  async function remove(id: number) {
    if (!confirm('Delete this tag?')) return
    await apiFetchRaw(`/api/v3/tag/${id}`, { method: 'DELETE' })
    load()
  }

  if (loading) return <PagePlaceholder title="Tags" description="Loading..." />
  if (error) return <PagePlaceholder title="Tags" description={`Error: ${error}`} />

  return (
    <PagePlaceholder
      title="Tags"
      description="Tags can be applied to series, indexers, download clients, notifications, import lists, release profiles, delay profiles, and auto-tagging rules."
    >
      <div style={{ marginBottom: 16 }}>
        <input
          value={newLabel}
          onChange={(e) => setNewLabel(e.target.value)}
          placeholder="New tag label"
          onKeyDown={(e) => { if (e.key === 'Enter') add() }}
          style={{ padding: 6, width: 220, marginRight: 8 }}
        />
        <button onClick={add}>Add</button>
      </div>
      {tags.length === 0 ? (
        <p>No tags yet.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: 8 }}>Label</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Series</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Indexers</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Download Clients</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Notifications</th>
              <th style={{ textAlign: 'left', padding: 8 }}></th>
            </tr>
          </thead>
          <tbody>
            {tags.map(t => (
              <tr key={t.id}>
                <td style={{ padding: 8 }}>{t.label}</td>
                <td style={{ padding: 8 }}>{t.seriesIds.length}</td>
                <td style={{ padding: 8 }}>{t.indexerIds.length}</td>
                <td style={{ padding: 8 }}>{t.downloadClientIds.length}</td>
                <td style={{ padding: 8 }}>{t.notificationIds.length}</td>
                <td style={{ padding: 8 }}><button onClick={() => remove(t.id)}>Delete</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PagePlaceholder>
  )
}
