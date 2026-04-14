import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SeriesEditor hosts /serieseditor — mass-edit view of the series library,
 * allowing bulk updates to quality profile, root folder, monitored status,
 * tags, and bulk delete. Pending UI; backend CRUD already supports this.
 */
export function SeriesEditor() {
  return (
    <PagePlaceholder
      title="Mass Editor"
      description="Bulk edit quality profile, root folder, monitored status, or tags across many series. UI pending; per-series PUT endpoints already exist."
    />
  )
}
