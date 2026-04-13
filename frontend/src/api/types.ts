export interface Series {
  id: number
  title: string
  sortTitle: string
  status: string
  seriesType: string
  tvdbId: number
  path: string
  monitored: boolean
  year: number
  statistics?: SeriesStatistics
}

export interface SeriesStatistics {
  episodeCount: number
  episodeFileCount: number
  sizeOnDisk: number
  percentOfEpisodes: number
}

export interface Episode {
  id: number
  seriesId: number
  seasonNumber: number
  episodeNumber: number
  title: string
  monitored: boolean
  hasFile: boolean
}

export interface Page<T> {
  data: T[]
  pagination: { limit: number; nextCursor?: string; hasMore: boolean }
}

export interface SystemStatus {
  appName: string
  version: string
  databaseType: string
  runtimeName: string
}
