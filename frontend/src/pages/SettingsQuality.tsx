import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsQuality renders /settings/quality, where users edit the size
 * min/max/preferred per quality definition (MB-per-minute for TV). The
 * 18 seeded quality definitions are read-only today; this page will
 * add PUT /api/v3/qualitydefinition/{id} support.
 */
export function SettingsQuality() {
  return (
    <PagePlaceholder
      title="Quality"
      description="Edit size min/max/preferred per quality definition. The 18 seeded qualities are shown here; editable persistence is pending the /api/v3/qualitydefinition PUT endpoint."
    />
  )
}
