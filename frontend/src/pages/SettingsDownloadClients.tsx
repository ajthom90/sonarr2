import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsDownloadClients hosts /settings/downloadclients. Backend CRUD is
 * live at /api/v3/downloadclient. Remote path mappings will surface here
 * once the editor UI is added.
 */
export function SettingsDownloadClients() {
  return (
    <PagePlaceholder
      title="Download Clients"
      description="Usenet and torrent client configuration. Remote Path Mappings are a sub-panel of this page in Sonarr; wiring pending."
    />
  )
}
