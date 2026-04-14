import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SystemUpdates hosts /system/updates — available GitHub releases with
 * changelog and install buttons. The update checker already polls the
 * Releases API and surfaces a health notice; this UI is pending.
 */
export function SystemUpdates() {
  return (
    <PagePlaceholder
      title="Updates"
      description="Available releases with changelog and install actions. sonarr2 checks GitHub Releases every 24h; the Docker/built-in/script/external update mechanism selector is pending."
    />
  )
}
