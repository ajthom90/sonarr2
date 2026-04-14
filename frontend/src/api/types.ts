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

export interface QualityProfileItemQuality {
  id: number
  name: string
  source: string
  resolution: string
}

export interface QualityProfileItem {
  quality: QualityProfileItemQuality
  items: unknown[]
  allowed: boolean
}

export interface QualityProfile {
  id: number
  name: string
  upgradeAllowed: boolean
  cutoff: number
  items: QualityProfileItem[]
  minFormatScore: number
  cutoffFormatScore: number
  formatItems: unknown[]
}

export interface QualityDefinition {
  id: number
  name: string
  source: string
  resolution: string
  minSize: number
  maxSize: number
  preferredSize: number
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
  seasonFolder?: boolean
  monitorNewItems?: 'all' | 'none'
  addOptions?: AddSeriesAddOptions
}

export interface GeneralSettings {
  apiKey: string
  authMode: string
  tvdbApiKey: string
}

export interface BackupInfo {
  name: string
  size: number
  time: string
}

export interface CustomFormatSpecField {
  name: string
  value: string
}

export interface CustomFormatSpec {
  name: string
  implementation: string
  negate: boolean
  required: boolean
  fields: CustomFormatSpecField[]
}

export interface CustomFormat {
  id: number
  name: string
  includeCustomFormatWhenRenaming: boolean
  specifications: CustomFormatSpec[]
}

export interface FilesystemEntry {
  type: 'folder' | 'file'
  name: string
  path: string
}

export interface FilesystemListing {
  parent: string
  directories: FilesystemEntry[]
  files: FilesystemEntry[]
}

export interface LibraryImportMatch {
  tvdbId: number
  title: string
  year: number
  overview?: string
}

export interface LibraryImportEntry {
  folderName: string
  relativePath: string
  absolutePath: string
  tvdbMatch: LibraryImportMatch | null
  alreadyImported: boolean
}

export interface CreateRootFolderRequest {
  path: string
}

export interface AddSeriesAddOptions {
  monitor?: 'all' | 'none' | 'future' | 'missing' | 'existing' | 'pilot' | 'firstSeason' | 'lastSeason'
  searchForMissingEpisodes?: boolean
  searchForCutoffUnmetEpisodes?: boolean
}

export interface ProviderFieldSchema {
  name: string
  label: string
  type: 'text' | 'password' | 'number' | 'checkbox' | 'select' | 'multiselect'
  required?: boolean
  default?: string
  placeholder?: string
  helpText?: string
  advanced?: boolean
}

export interface ProviderSchema {
  implementation: string
  name: string
  fields: ProviderFieldSchema[]
}

export interface IndexerResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  enableRss: boolean
  enableAutomaticSearch: boolean
  enableInteractiveSearch: boolean
  priority: number
  added?: string
}

export interface DownloadClientResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  enable: boolean
  priority: number
  added?: string
}

export interface RemotePathMapping {
  id: number
  host: string
  remotePath: string
  localPath: string
}

export interface NotificationResource {
  id: number
  name: string
  implementation: string
  fields: Record<string, unknown>
  onGrab: boolean
  onDownload: boolean
  onHealthIssue: boolean
  tags: number[]
  added?: string
}

export interface WantedEpisodeV3 {
  id: number
  seriesId: number
  tvdbId: number
  episodeFileId: number
  seasonNumber: number
  episodeNumber: number
  absoluteEpisodeNumber?: number | null
  title: string
  airDate: string
  airDateUtc: string
  overview: string
  hasFile: boolean
  monitored: boolean
  runtime: number
}

export interface PagedEpisodeResponse {
  page: number
  pageSize: number
  sortKey: string
  sortDirection: string
  totalRecords: number
  records: WantedEpisodeV3[]
}
