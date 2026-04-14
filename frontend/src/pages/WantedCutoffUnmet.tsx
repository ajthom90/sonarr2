import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * WantedCutoffUnmet is the /wanted/cutoffunmet page, listing episodes that
 * have a file below the quality profile's Cutoff. Wire target:
 * /api/v3/wanted/cutoff (to be added).
 */
export function WantedCutoffUnmet() {
  return (
    <PagePlaceholder
      title="Cutoff Unmet"
      description="Episodes that have a file but below the quality profile's cutoff. These will be upgraded when a better release appears. Backend endpoint /api/v3/wanted/cutoff is pending."
    />
  )
}
