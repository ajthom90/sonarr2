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
  airDate?: string
  airDateUtc?: string
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

export interface Command {
  id: number
  name: string
  status: string
  queued: string
  started?: string
  ended?: string
  trigger: string
}

export interface HistoryEntry {
  id: number
  seriesId: number
  episodeId: number
  sourceTitle: string
  eventType: string
  date: string
}

export interface HealthItem {
  source: string
  type: string
  message: string
  wikiUrl?: string
}

export interface WantedEpisode {
  id: number
  seriesId: number
  seasonNumber: number
  episodeNumber: number
  title: string
  airDate?: string
  series?: { title: string }
}

export interface Indexer {
  id: number
  name: string
  implementation: string
  enableRss: boolean
  enableAutomaticSearch: boolean
  priority: number
}

export interface DownloadClient {
  id: number
  name: string
  implementation: string
  enable: boolean
  priority: number
}

export interface QualityProfile {
  id: number
  name: string
  upgradeAllowed: boolean
  items: Array<{ qualityId: number; allowed: boolean }>
}

export interface SeriesLookupResult {
  tvdbId: number
  title: string
  year: number
  overview: string
  status: string
  network: string
  titleSlug: string
}

export interface RootFolder {
  id: number
  path: string
  freeSpace: number
  accessible: boolean
}

export interface AddSeriesRequest {
  title: string
  tvdbId: number
  titleSlug: string
  path: string
  qualityProfileId: number
  monitored: boolean
  seriesType: string
  status: string
}
