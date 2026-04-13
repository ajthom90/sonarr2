import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSeriesLookup, useRootFolders, useQualityProfiles, useAddSeries } from '../api/hooks'
import type { SeriesLookupResult } from '../api/types'
import styles from './AddSeries.module.css'

export function AddSeries() {
  const navigate = useNavigate()
  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedTerm, setDebouncedTerm] = useState('')
  const [selected, setSelected] = useState<SeriesLookupResult | null>(null)
  const [rootFolder, setRootFolder] = useState('')
  const [customRootFolder, setCustomRootFolder] = useState('')
  const [qualityProfileId, setQualityProfileId] = useState<number>(0)
  const [monitored, setMonitored] = useState(true)
  const [seriesType, setSeriesType] = useState('standard')

  const { data: results, isLoading: searching } = useSeriesLookup(debouncedTerm)
  const { data: rootFolders } = useRootFolders()
  const { data: profilesPage } = useQualityProfiles()
  const profiles = profilesPage?.data
  const addSeries = useAddSeries()

  // Debounce search input
  const [debounceTimer, setDebounceTimer] = useState<ReturnType<typeof setTimeout> | null>(null)
  function handleSearchChange(value: string) {
    setSearchTerm(value)
    if (debounceTimer) clearTimeout(debounceTimer)
    setDebounceTimer(setTimeout(() => setDebouncedTerm(value), 300))
  }

  // Set defaults when root folders or profiles load
  if (rootFolders && rootFolders.length > 0 && !rootFolder && rootFolders[0]) {
    setRootFolder(rootFolders[0].path)
  }
  if (profiles && profiles.length > 0 && qualityProfileId === 0 && profiles[0]) {
    setQualityProfileId(profiles[0].id)
  }

  function handleAdd(e: FormEvent) {
    e.preventDefault()
    if (!selected) return

    const effectiveRoot = rootFolder === '__custom__' ? customRootFolder : rootFolder
    if (!effectiveRoot) return

    addSeries.mutate({
      title: selected.title,
      tvdbId: selected.tvdbId,
      titleSlug: selected.titleSlug,
      path: `${effectiveRoot}/${selected.title} (${selected.year})`,
      qualityProfileId,
      monitored,
      seriesType,
      status: selected.status || 'continuing',
    }, {
      onSuccess: () => navigate('/'),
    })
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Add New Series</h1>

      <div className={styles.searchBox}>
        <input
          type="text"
          placeholder="Search for a series by name or tvdb:####"
          value={searchTerm}
          onChange={(e) => handleSearchChange(e.target.value)}
          className={styles.searchInput}
          autoFocus
        />
      </div>

      {searching && <p className={styles.hint}>Searching...</p>}

      {!selected && results && results.length > 0 && (
        <div className={styles.results}>
          {results.map((r) => (
            <button
              key={r.tvdbId}
              className={styles.resultCard}
              onClick={() => setSelected(r)}
            >
              <div className={styles.resultTitle}>
                {r.title} {r.year > 0 && <span className={styles.resultYear}>({r.year})</span>}
              </div>
              <div className={styles.resultMeta}>
                {r.network && <span>{r.network}</span>}
                {r.status && <span>{r.status}</span>}
              </div>
              {r.overview && (
                <div className={styles.resultOverview}>{r.overview.slice(0, 200)}{r.overview.length > 200 ? '...' : ''}</div>
              )}
            </button>
          ))}
        </div>
      )}

      {!selected && results && results.length === 0 && debouncedTerm.length >= 2 && !searching && (
        <p className={styles.hint}>No results found for &ldquo;{debouncedTerm}&rdquo;</p>
      )}

      {selected && (
        <div className={styles.addForm}>
          <div className={styles.selectedInfo}>
            <h2>{selected.title} {selected.year > 0 && `(${selected.year})`}</h2>
            <p className={styles.resultOverview}>{selected.overview}</p>
            <button className={styles.changeBtn} onClick={() => setSelected(null)}>Change Selection</button>
          </div>

          <form onSubmit={handleAdd} className={styles.form}>
            <label className={styles.label}>
              Root Folder
              <select
                value={rootFolder}
                onChange={(e) => setRootFolder(e.target.value)}
                className={styles.select}
              >
                {rootFolders?.map((f) => (
                  <option key={f.id} value={f.path}>{f.path}</option>
                ))}
                <option value="__custom__">Add a new path...</option>
              </select>
            </label>

            {rootFolder === '__custom__' && (
              <label className={styles.label}>
                Custom Root Folder Path
                <input
                  type="text"
                  value={customRootFolder}
                  onChange={(e) => setCustomRootFolder(e.target.value)}
                  className={styles.input}
                  placeholder="/path/to/tv"
                  required
                />
              </label>
            )}

            <label className={styles.label}>
              Quality Profile
              <select
                value={qualityProfileId}
                onChange={(e) => setQualityProfileId(Number(e.target.value))}
                className={styles.select}
              >
                {profiles?.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </label>

            <label className={styles.label}>
              Series Type
              <select
                value={seriesType}
                onChange={(e) => setSeriesType(e.target.value)}
                className={styles.select}
              >
                <option value="standard">Standard</option>
                <option value="daily">Daily</option>
                <option value="anime">Anime</option>
              </select>
            </label>

            <label className={styles.checkboxLabel}>
              <input
                type="checkbox"
                checked={monitored}
                onChange={(e) => setMonitored(e.target.checked)}
              />
              Monitored
            </label>

            {addSeries.isError && (
              <div className={styles.error}>
                Failed to add series: {addSeries.error instanceof Error ? addSeries.error.message : 'Unknown error'}
              </div>
            )}

            <button type="submit" className={styles.addButton} disabled={addSeries.isPending}>
              {addSeries.isPending ? 'Adding...' : `Add ${selected.title}`}
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
