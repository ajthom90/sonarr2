import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsIndexers is the dedicated /settings/indexers page. Indexer CRUD is
 * available today via /api/v3/indexer and is reachable via the combined
 * Settings UI; this route will host the full-page editor.
 */
export function SettingsIndexers() {
  return (
    <PagePlaceholder
      title="Indexers"
      description="Configure Newznab, Torznab, and tracker-specific indexers. Backend CRUD is live at /api/v3/indexer; the dedicated page UI is pending."
    />
  )
}
