import { useEffect, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { Link } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { ApiError } from '../api/client'
import { apiV3 } from '../api/v3'
import {
  useRootFolders,
  useCreateRootFolder,
  useLibraryImportScan,
  useQualityProfiles,
  useAddSeriesV3,
} from '../api/hooks'
import type {
  LibraryImportEntry,
  LibraryImportMatch,
  QualityProfile,
  RootFolder,
  SeriesLookupResult,
} from '../api/types'
import { FileBrowserModal } from '../components/FileBrowserModal'
import { SearchOverrideModal } from '../components/SearchOverrideModal'
import styles from './LibraryImport.module.css'

type MonitorMode =
  | 'all'
  | 'none'
  | 'future'
  | 'missing'
  | 'existing'
  | 'pilot'
  | 'firstSeason'
  | 'lastSeason'

type SeriesType = 'standard' | 'daily' | 'anime'

type DirtyFields = {
  monitor?: true
  qualityProfileId?: true
  seriesType?: true
  seasonFolder?: true
}

type RowState = {
  entry: LibraryImportEntry
  tvdbMatch: LibraryImportMatch | null
  checked: boolean
  monitor: MonitorMode
  qualityProfileId: number
  seriesType: SeriesType
  seasonFolder: boolean
  dirty: DirtyFields
  status: 'pending' | 'creating' | 'created' | 'failed'
  error?: string
}

function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export function LibraryImport() {
  const qc = useQueryClient()
  const { data: rootFoldersRaw } = useRootFolders()
  const rootFolders = rootFoldersRaw ?? []
  const createRF = useCreateRootFolder()
  const [browserOpen, setBrowserOpen] = useState(false)
  const [selectedRFId, setSelectedRFId] = useState<number | null>(null)

  async function pickFolder(path: string) {
    setBrowserOpen(false)
    // Try local cache first — the happy path when the root folder list is
    // already loaded in React Query.
    const existing = rootFolders.find((rf) => rf.path === path)
    if (existing) {
      setSelectedRFId(existing.id)
      return
    }
    try {
      const created = (await createRF.mutateAsync({ path })) as RootFolder
      setSelectedRFId(created.id)
    } catch (err) {
      // 409 means the server already has a root folder at this path but our
      // local cache didn't know — likely the user added it earlier, got a
      // scan error (e.g. missing TVDB key), navigated away, and came back.
      // Re-fetch the list and adopt the existing row instead of erroring.
      if (err instanceof ApiError && err.status === 409) {
        try {
          const fresh = await qc.fetchQuery<RootFolder[]>({
            queryKey: ['rootfolders'],
            queryFn: () => apiV3.get<RootFolder[]>('/rootfolder'),
          })
          const match = fresh.find((rf) => rf.path === path)
          if (match) {
            setSelectedRFId(match.id)
            return
          }
        } catch {
          // fall through to alert below
        }
      }
      alert(`Failed to add root folder: ${(err as Error).message}`)
    }
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Library Import</h1>
      {selectedRFId === null ? (
        <EmptyState
          existing={rootFolders}
          onChoose={() => setBrowserOpen(true)}
          onPickExisting={setSelectedRFId}
        />
      ) : (
        <ScanAndGrid
          rootFolderId={selectedRFId}
          onChangeRootFolder={() => setSelectedRFId(null)}
        />
      )}
      <FileBrowserModal
        isOpen={browserOpen}
        initialPath="/"
        onSelect={pickFolder}
        onCancel={() => setBrowserOpen(false)}
      />
    </div>
  )
}

function EmptyState({
  existing,
  onChoose,
  onPickExisting,
}: {
  existing: RootFolder[]
  onChoose: () => void
  onPickExisting: (id: number) => void
}) {
  return (
    <div className={styles.empty}>
      <section className={styles.tips}>
        <h2>Tips for a smooth import</h2>
        <ul>
          <li>
            Ensure quality is in filenames (e.g. <code>episode.s01e01.bluray.mkv</code>).
          </li>
          <li>
            Point to the <em>parent</em> folder — e.g. <code>/data/tv</code>, not{' '}
            <code>/data/tv/The Simpsons</code>. Each series must be in its own sub-folder.
          </li>
          <li>Don&apos;t use this for downloads; it expects an organized library.</li>
        </ul>
      </section>
      <div className={styles.emptyActions}>
        <button className={styles.primary} onClick={onChoose}>
          Choose folder to import from
        </button>
        {existing.length > 0 && (
          <div className={styles.existing}>
            Or reuse an existing root folder:
            <select
              onChange={(e) => onPickExisting(Number(e.target.value))}
              defaultValue=""
            >
              <option value="" disabled>
                (select)
              </option>
              {existing.map((rf) => (
                <option key={rf.id} value={rf.id}>
                  {rf.path}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>
    </div>
  )
}

function ScanAndGrid({
  rootFolderId,
  onChangeRootFolder,
}: {
  rootFolderId: number
  onChangeRootFolder: () => void
}) {
  const scan = useLibraryImportScan(rootFolderId)
  const { data: qpsPage } = useQualityProfiles()
  const addSeries = useAddSeriesV3()
  const qps: QualityProfile[] = qpsPage?.data ?? []

  const [rows, setRows] = useState<RowState[]>([])
  const [bulkMonitor, setBulkMonitor] = useState<MonitorMode>('all')
  const [bulkQP, setBulkQP] = useState<number>(qps[0]?.id ?? 1)
  const [bulkSeriesType, setBulkSeriesType] = useState<SeriesType>('standard')
  const [bulkSeasonFolder, setBulkSeasonFolder] = useState<boolean>(true)
  const [importStatus, setImportStatus] = useState<'idle' | 'importing' | 'done'>('idle')
  const [overrideForIdx, setOverrideForIdx] = useState<number | null>(null)

  // Initialize bulkQP as soon as profiles arrive.
  useEffect(() => {
    if (qps.length > 0) {
      const firstId = qps[0]?.id
      if (firstId !== undefined && (bulkQP === 1 || !qps.find((q) => q.id === bulkQP))) {
        setBulkQP(firstId)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [qpsPage])

  // Seed rows on scan success.
  useEffect(() => {
    if (!scan.data) return
    setRows(
      scan.data.map((e) => ({
        entry: e,
        tvdbMatch: e.tvdbMatch,
        checked: e.tvdbMatch !== null && !e.alreadyImported,
        monitor: bulkMonitor,
        qualityProfileId: bulkQP,
        seriesType: bulkSeriesType,
        seasonFolder: bulkSeasonFolder,
        dirty: {},
        status: 'pending',
      })),
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scan.data])

  // Cascade bulk defaults to non-dirty rows.
  useEffect(() => {
    setRows((rs) =>
      rs.map((r) => ({
        ...r,
        monitor: r.dirty.monitor ? r.monitor : bulkMonitor,
        qualityProfileId: r.dirty.qualityProfileId ? r.qualityProfileId : bulkQP,
        seriesType: r.dirty.seriesType ? r.seriesType : bulkSeriesType,
        seasonFolder: r.dirty.seasonFolder ? r.seasonFolder : bulkSeasonFolder,
      })),
    )
  }, [bulkMonitor, bulkQP, bulkSeriesType, bulkSeasonFolder])

  const selected = rows.filter((r) => r.checked && r.tvdbMatch && !r.entry.alreadyImported)
  const unmatched = rows.filter((r) => !r.tvdbMatch && !r.entry.alreadyImported)
  const already = rows.filter((r) => r.entry.alreadyImported)

  async function handleImport() {
    setImportStatus('importing')
    for (let i = 0; i < rows.length; i++) {
      const row = rows[i]
      if (!row) continue
      if (!row.checked || !row.tvdbMatch || row.entry.alreadyImported) continue
      const match = row.tvdbMatch
      setRows((rs) => rs.map((r, idx) => (idx === i ? { ...r, status: 'creating' } : r)))
      try {
        await addSeries.mutateAsync({
          title: match.title,
          tvdbId: match.tvdbId,
          titleSlug: slugify(match.title),
          path: row.entry.absolutePath,
          qualityProfileId: row.qualityProfileId,
          seriesType: row.seriesType,
          seasonFolder: row.seasonFolder,
          monitorNewItems: 'all',
          monitored: true,
          status: 'continuing',
          addOptions: { monitor: row.monitor, searchForMissingEpisodes: false },
        })
        setRows((rs) => rs.map((r, idx) => (idx === i ? { ...r, status: 'created' } : r)))
      } catch (err) {
        const msg = (err as Error).message
        setRows((rs) =>
          rs.map((r, idx) => (idx === i ? { ...r, status: 'failed', error: msg } : r)),
        )
      }
    }
    setImportStatus('done')
  }

  if (scan.isLoading) {
    return <div className={styles.scanning}>Scanning…</div>
  }

  if (scan.isError) {
    const err = scan.error
    const fixPath =
      err instanceof ApiError && typeof err.details.fixPath === 'string'
        ? err.details.fixPath
        : undefined
    const message = err instanceof Error ? err.message : 'Scan failed'
    return (
      <div className={styles.page}>
        <div className={styles.banner}>
          <span>{message}</span>
          {fixPath && (
            <Link to={fixPath} className={styles.bannerAction}>
              Configure
            </Link>
          )}
          <button className={styles.bannerAction} onClick={onChangeRootFolder}>
            Choose a different folder
          </button>
        </div>
      </div>
    )
  }

  if (rows.length === 0) {
    return (
      <div>
        <p className={styles.status}>
          No importable folders found.{' '}
          <button className={styles.linkBtn} onClick={onChangeRootFolder}>
            Choose a different folder
          </button>
        </p>
      </div>
    )
  }

  const overrideRow = overrideForIdx !== null ? rows[overrideForIdx] : undefined

  return (
    <div className={styles.grid}>
      <div className={styles.bulk}>
        <label>
          Monitor
          <select
            value={bulkMonitor}
            onChange={(e) => setBulkMonitor(e.target.value as MonitorMode)}
          >
            <option value="all">All Episodes</option>
            <option value="future">Future Episodes</option>
            <option value="missing">Missing Episodes</option>
            <option value="existing">Existing Episodes</option>
            <option value="pilot">Pilot Episode</option>
            <option value="firstSeason">First Season</option>
            <option value="lastSeason">Last Season</option>
            <option value="none">None</option>
          </select>
        </label>
        <label>
          Quality Profile
          <select value={bulkQP} onChange={(e) => setBulkQP(Number(e.target.value))}>
            {qps.map((qp) => (
              <option key={qp.id} value={qp.id}>
                {qp.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Series Type
          <select
            value={bulkSeriesType}
            onChange={(e) => setBulkSeriesType(e.target.value as SeriesType)}
          >
            <option value="standard">Standard</option>
            <option value="daily">Daily</option>
            <option value="anime">Anime</option>
          </select>
        </label>
        <label>
          Season Folder
          <input
            type="checkbox"
            checked={bulkSeasonFolder}
            onChange={(e) => setBulkSeasonFolder(e.target.checked)}
          />
        </label>
        <button className={styles.linkBtn} onClick={onChangeRootFolder}>
          Change folder
        </button>
      </div>

      <table className={styles.table}>
        <thead>
          <tr>
            <th></th>
            <th>Folder</th>
            <th>Matched Series</th>
            <th>Monitor</th>
            <th>Profile</th>
            <th>Type</th>
            <th>SF</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <Row
              key={row.entry.absolutePath}
              row={row}
              i={i}
              qps={qps}
              setRows={setRows}
              onChangeMatch={setOverrideForIdx}
            />
          ))}
        </tbody>
      </table>

      <footer className={styles.footer}>
        <div>
          {selected.length} selected · {unmatched.length} unmatched · {already.length} already
          imported
        </div>
        <button
          className={styles.primary}
          disabled={selected.length === 0 || importStatus === 'importing'}
          onClick={handleImport}
        >
          Import {selected.length} Series
        </button>
      </footer>

      {importStatus === 'done' && (
        <div className={styles.banner}>
          Imported {rows.filter((r) => r.status === 'created').length} succeeded,{' '}
          {rows.filter((r) => r.status === 'failed').length} failed.{' '}
          <Link to="/">Back to series</Link>
        </div>
      )}

      {overrideForIdx !== null && overrideRow && (
        <SearchOverrideModal
          isOpen
          initialTerm={overrideRow.entry.folderName}
          onSelect={(m: SeriesLookupResult | null) => {
            const capturedIdx = overrideForIdx
            setRows((rs) =>
              rs.map((r, idx) =>
                idx === capturedIdx
                  ? {
                      ...r,
                      tvdbMatch: m
                        ? {
                            tvdbId: m.tvdbId,
                            title: m.title,
                            year: m.year,
                            overview: m.overview,
                          }
                        : null,
                      checked: !!m,
                    }
                  : r,
              ),
            )
            setOverrideForIdx(null)
          }}
          onCancel={() => setOverrideForIdx(null)}
        />
      )}
    </div>
  )
}

function Row({
  row,
  i,
  qps,
  setRows,
  onChangeMatch,
}: {
  row: RowState
  i: number
  qps: QualityProfile[]
  setRows: Dispatch<SetStateAction<RowState[]>>
  onChangeMatch: (idx: number) => void
}) {
  function update<K extends keyof RowState>(k: K, v: RowState[K]) {
    setRows((rs) =>
      rs.map((r, idx) =>
        idx === i ? { ...r, [k]: v, dirty: { ...r.dirty, [k]: true } } : r,
      ),
    )
  }
  const disabled = row.entry.alreadyImported
  return (
    <tr className={row.status === 'failed' ? styles.rowFailed : undefined}>
      <td>
        <input
          type="checkbox"
          checked={row.checked}
          disabled={disabled || !row.tvdbMatch}
          onChange={(e) =>
            setRows((rs) =>
              rs.map((r, idx) => (idx === i ? { ...r, checked: e.target.checked } : r)),
            )
          }
        />
      </td>
      <td>
        {row.entry.folderName}
        {disabled && <span className={styles.lock}> Already in library</span>}
      </td>
      <td>
        {row.tvdbMatch ? (
          <span>
            {row.tvdbMatch.title} ({row.tvdbMatch.year}){' '}
            <button type="button" className={styles.linkBtn} onClick={() => onChangeMatch(i)}>
              Change
            </button>
          </span>
        ) : (
          <span className={styles.noMatch}>
            No match —{' '}
            <button type="button" className={styles.linkBtn} onClick={() => onChangeMatch(i)}>
              Change
            </button>
          </span>
        )}
      </td>
      <td>
        <select
          value={row.monitor}
          onChange={(e) => update('monitor', e.target.value as MonitorMode)}
          disabled={disabled}
        >
          <option value="all">All</option>
          <option value="future">Future</option>
          <option value="missing">Missing</option>
          <option value="existing">Existing</option>
          <option value="pilot">Pilot</option>
          <option value="firstSeason">First Season</option>
          <option value="lastSeason">Last Season</option>
          <option value="none">None</option>
        </select>
      </td>
      <td>
        <select
          value={row.qualityProfileId}
          onChange={(e) => update('qualityProfileId', Number(e.target.value))}
          disabled={disabled}
        >
          {qps.map((qp) => (
            <option key={qp.id} value={qp.id}>
              {qp.name}
            </option>
          ))}
        </select>
      </td>
      <td>
        <select
          value={row.seriesType}
          onChange={(e) => update('seriesType', e.target.value as SeriesType)}
          disabled={disabled}
        >
          <option value="standard">Standard</option>
          <option value="daily">Daily</option>
          <option value="anime">Anime</option>
        </select>
      </td>
      <td>
        <input
          type="checkbox"
          checked={row.seasonFolder}
          onChange={(e) => update('seasonFolder', e.target.checked)}
          disabled={disabled}
        />
      </td>
      <td>
        {row.status === 'creating' && <span>…</span>}
        {row.status === 'created' && <span className={styles.ok}>✓</span>}
        {row.status === 'failed' && (
          <span className={styles.err} title={row.error}>
            ✗
          </span>
        )}
      </td>
    </tr>
  )
}
