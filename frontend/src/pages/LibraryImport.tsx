import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * LibraryImport hosts /add/import — mass-import of existing folder contents
 * by scanning a root folder and matching folders to series. Pending.
 */
export function LibraryImport() {
  return (
    <PagePlaceholder
      title="Library Import"
      description="Import an existing folder of series by scanning and matching to TheTVDB. Pending manual-import endpoint and matching flow."
    />
  )
}
