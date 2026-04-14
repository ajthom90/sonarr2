import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsMetadata hosts /settings/metadata — file-emitting metadata
 * consumers (Kodi/XBMC, Plex, Roksbox, WDTV). Subsystem pending.
 *
 * NOTE: This page is separate from /settings/metadatasource. Metadata
 * consumers WRITE nfo/image files for local media players; the metadata
 * source (TheTVDB) is the backing catalog.
 */
export function SettingsMetadata() {
  return (
    <PagePlaceholder
      title="Metadata"
      description="Emit .nfo and image files for Kodi/XBMC, Plex, Roksbox, or WDTV. Subsystem and providers are pending."
    />
  )
}
