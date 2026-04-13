import { useQuery } from '@tanstack/react-query'
import { api } from './client'
import type { Series, Episode, Page, SystemStatus } from './types'

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

export function useSystemStatus() {
  return useQuery({
    queryKey: ['system', 'status'],
    queryFn: () => api.get<SystemStatus>('/system/status'),
  })
}
