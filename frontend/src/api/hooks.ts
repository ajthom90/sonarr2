import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type { Series, Episode, Page, SystemStatus, Command, HistoryEntry, HealthItem, WantedEpisode, Indexer, DownloadClient, QualityProfile, SeriesLookupResult, RootFolder, AddSeriesRequest } from './types'

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
