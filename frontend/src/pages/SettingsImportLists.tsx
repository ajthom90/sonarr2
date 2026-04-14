import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsImportLists hosts /settings/importlists. Import lists (AniList,
 * MAL, Plex Watchlist, Trakt, generic RSS, Simkl, Sonarr-to-Sonarr) are
 * pulled in on a 5-minute cadence to auto-add series. Subsystem pending.
 */
export function SettingsImportLists() {
  return (
    <PagePlaceholder
      title="Import Lists"
      description="Auto-populate your series library from AniList, MyAnimeList, Plex Watchlist, Trakt, generic RSS, Simkl, or another Sonarr instance. Subsystem and providers are pending."
    />
  )
}
