import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import { apiV3 } from './v3'
import type { Series, Episode, Page, SystemStatus, Command, HistoryEntry, HealthItem, WantedEpisode, Indexer, DownloadClient, QualityProfile, QualityDefinition, SeriesLookupResult, RootFolder, AddSeriesRequest, GeneralSettings, CustomFormat, BackupInfo, FilesystemListing, LibraryImportEntry, CreateRootFolderRequest, ProviderSchema, IndexerResource, DownloadClientResource, RemotePathMapping } from './types'

export function useSeriesList() {
  return useQuery({
    queryKey: ['series'],
    queryFn: () => api.get<Page<Series>>('/series'),
  })
}

export function useSeries(id: number) {
  return useQuery({
    queryKey: ['series', id],
    queryFn: () => api.get<Series>(`/series/${id}`),
    enabled: id > 0,
  })
}

export function useEpisodes(seriesId: number) {
  return useQuery({
    queryKey: ['episodes', seriesId],
    queryFn: () => api.get<Page<Episode>>(`/episode?seriesId=${seriesId}`),
    enabled: seriesId > 0,
  })
}

export function useCalendar(start: string, end: string) {
  return useQuery({
    queryKey: ['calendar', start, end],
    queryFn: () => api.get<Page<Episode>>(`/calendar?start=${start}&end=${end}`),
  })
}

export function useCommands() {
  return useQuery({
    queryKey: ['commands'],
    queryFn: () => api.get<Page<Command>>('/command'),
    refetchInterval: 5000,
  })
}

export function useHistory(cursor?: string) {
  return useQuery({
    queryKey: ['history', cursor],
    queryFn: () => api.get<Page<HistoryEntry>>(`/history${cursor ? `?cursor=${cursor}` : ''}`),
  })
}

export function useWantedMissing(cursor?: string) {
  return useQuery({
    queryKey: ['wanted', 'missing', cursor],
    queryFn: () => api.get<Page<WantedEpisode>>(`/wanted/missing${cursor ? `?cursor=${cursor}` : ''}`),
  })
}

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => api.get<HealthItem[]>('/health'),
  })
}

export function useSystemStatus() {
  return useQuery({
    queryKey: ['system', 'status'],
    queryFn: () => api.get<SystemStatus>('/system/status'),
  })
}

export function useIndexers() {
  return useQuery({
    queryKey: ['indexers'],
    queryFn: () => api.get<Page<Indexer>>('/indexer'),
  })
}

export function useDownloadClients() {
  return useQuery({
    queryKey: ['downloadclients'],
    queryFn: () => api.get<Page<DownloadClient>>('/downloadclient'),
  })
}

export function useQualityProfiles() {
  return useQuery({
    queryKey: ['qualityprofiles'],
    queryFn: () => api.get<Page<QualityProfile>>('/qualityprofile'),
  })
}

export function useQualityDefinitions() {
  return useQuery({
    queryKey: ['qualitydefinitions'],
    queryFn: () => api.get<QualityDefinition[]>('/qualitydefinition'),
  })
}

export function useCreateQualityProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<QualityProfile, 'id'>) =>
      api.post<QualityProfile>('/qualityprofile', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['qualityprofiles'] }),
  })
}

export function useUpdateQualityProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: QualityProfile) =>
      api.put<QualityProfile>(`/qualityprofile/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['qualityprofiles'] }),
  })
}

export function useDeleteQualityProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete(`/qualityprofile/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['qualityprofiles'] }),
  })
}

export function useDeleteIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete(`/indexer/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['indexers'] }),
  })
}

export function useAddIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: unknown) => api.post('/indexer', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['indexers'] }),
  })
}

export function useDeleteDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete(`/downloadclient/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['downloadclients'] }),
  })
}

export function useAddDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: unknown) => api.post('/downloadclient', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['downloadclients'] }),
  })
}

export function useSeriesLookup(term: string) {
  return useQuery({
    queryKey: ['series-lookup', term],
    queryFn: () => api.get<SeriesLookupResult[]>(`/series/lookup?term=${encodeURIComponent(term)}`),
    enabled: term.length >= 2,
  })
}

export function useRootFolders() {
  return useQuery({
    queryKey: ['rootfolders'],
    queryFn: () => api.get<RootFolder[]>('/rootfolder'),
  })
}

export function useAddSeries() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (series: AddSeriesRequest) => api.post<unknown>('/series', series),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

export function useUpdateSeries() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number } & Partial<Series>) =>
      api.put<Series>(`/series/${id}`, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['series', variables.id] })
      qc.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

export function useDeleteSeries() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, deleteFiles }: { id: number; deleteFiles: boolean }) =>
      api.delete(`/series/${id}${deleteFiles ? '?deleteFiles=true' : ''}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

export function useGeneralSettings() {
  return useQuery({
    queryKey: ['settings-general'],
    queryFn: () => api.get<GeneralSettings>('/config/general'),
  })
}

export function useUpdateGeneralSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (settings: Partial<GeneralSettings>) => api.put<GeneralSettings>('/config/general', settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings-general'] })
    },
  })
}

export function useCustomFormats() {
  return useQuery({
    queryKey: ['customformats'],
    queryFn: () => api.get<CustomFormat[]>('/customformat'),
  })
}

export function useCreateCustomFormat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<CustomFormat, 'id'>) =>
      api.post<CustomFormat>('/customformat', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['customformats'] }),
  })
}

export function useUpdateCustomFormat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: CustomFormat) =>
      api.put<CustomFormat>(`/customformat/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['customformats'] }),
  })
}

export function useDeleteCustomFormat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete(`/customformat/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['customformats'] }),
  })
}

export function useConnectionStatus() {
  return useQuery({
    queryKey: ['ping'],
    queryFn: async () => {
      const res = await fetch('/ping')
      return res.ok
    },
    refetchInterval: 30_000,
    retry: false,
  })
}

export function useUpdateEpisode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, monitored }: { id: number; monitored: boolean }) =>
      api.put<Episode>(`/episode/${id}`, { monitored }),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['episodes'] })
      qc.invalidateQueries({ queryKey: ['series', variables.id] })
    },
  })
}

export function useTriggerCommand() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (cmd: { name: string; body?: Record<string, unknown> }) =>
      api.post<Command>('/command', cmd),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['commands'] })
    },
  })
}

export function useBackups() {
  return useQuery({
    queryKey: ['system', 'backups'],
    queryFn: () => api.get<BackupInfo[]>('/system/backup'),
  })
}

export function useFilesystem(path: string) {
  return useQuery({
    queryKey: ['v3', 'filesystem', path],
    queryFn: () => apiV3.get<FilesystemListing>(`/filesystem?path=${encodeURIComponent(path)}`),
    enabled: path.length > 0,
  })
}

export function useCreateRootFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateRootFolderRequest) => apiV3.post<RootFolder>('/rootfolder', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['rootfolders'] })
      qc.invalidateQueries({ queryKey: ['v3', 'rootfolder'] })
    },
  })
}

export function useDeleteRootFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/rootfolder/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['rootfolders'] })
      qc.invalidateQueries({ queryKey: ['v3', 'rootfolder'] })
    },
  })
}

export function useLibraryImportScan(
  rootFolderId: number,
  opts?: { previewOnly?: boolean; enabled?: boolean },
) {
  const preview = opts?.previewOnly ? '&previewOnly=true' : ''
  return useQuery({
    queryKey: ['v3', 'libraryimport', rootFolderId, opts?.previewOnly ?? false],
    queryFn: () =>
      apiV3.get<LibraryImportEntry[]>(
        `/libraryimport/scan?rootFolderId=${rootFolderId}${preview}`,
      ),
    enabled: (opts?.enabled ?? true) && rootFolderId > 0,
    staleTime: 60_000,
  })
}

export function useLibraryImportPreview(rootFolderId: number) {
  return useLibraryImportScan(rootFolderId, { previewOnly: true })
}

export function useAddSeriesV3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: AddSeriesRequest) => apiV3.post<unknown>('/series', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

// ── Provider settings (sub-project #2) ───────────────────────────────────────

export function useIndexerSchema() {
  return useQuery({
    queryKey: ['v3', 'indexer', 'schema'],
    queryFn: () => apiV3.get<ProviderSchema[]>('/indexer/schema'),
  })
}

export function useIndexersV3() {
  return useQuery({
    queryKey: ['v3', 'indexer'],
    queryFn: () => apiV3.get<IndexerResource[]>('/indexer'),
  })
}

export function useCreateIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<IndexerResource, 'id' | 'added'>) =>
      apiV3.post<IndexerResource>('/indexer', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

export function useUpdateIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: IndexerResource) =>
      apiV3.put<IndexerResource>(`/indexer/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

export function useDeleteIndexerV3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/indexer/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'indexer'] }),
  })
}

export function useDownloadClientSchema() {
  return useQuery({
    queryKey: ['v3', 'downloadclient', 'schema'],
    queryFn: () => apiV3.get<ProviderSchema[]>('/downloadclient/schema'),
  })
}

export function useDownloadClientsV3() {
  return useQuery({
    queryKey: ['v3', 'downloadclient'],
    queryFn: () => apiV3.get<DownloadClientResource[]>('/downloadclient'),
  })
}

export function useCreateDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<DownloadClientResource, 'id' | 'added'>) =>
      apiV3.post<DownloadClientResource>('/downloadclient', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

export function useUpdateDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: DownloadClientResource) =>
      apiV3.put<DownloadClientResource>(`/downloadclient/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

export function useDeleteDownloadClientV3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/downloadclient/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'downloadclient'] }),
  })
}

export function useRemotePathMappings() {
  return useQuery({
    queryKey: ['v3', 'remotepathmapping'],
    queryFn: () => apiV3.get<RemotePathMapping[]>('/remotepathmapping'),
  })
}

export function useCreateRemotePathMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<RemotePathMapping, 'id'>) =>
      apiV3.post<RemotePathMapping>('/remotepathmapping', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'remotepathmapping'] }),
  })
}

export function useDeleteRemotePathMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiV3.delete(`/remotepathmapping/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['v3', 'remotepathmapping'] }),
  })
}
