import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SystemLogFiles hosts /system/logs/files — list of on-disk rotated logs
 * with download/view actions.
 */
export function SystemLogFiles() {
  return (
    <PagePlaceholder
      title="Log Files"
      description="On-disk rotated log files. Listing, download, and in-browser view endpoints (/api/v3/log/file) are pending."
    />
  )
}
