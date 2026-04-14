import { SettingsGeneral } from './SettingsGeneral'

/**
 * SettingsMetadataSource hosts /settings/metadatasource. The TVDB API key
 * currently lives on the General settings page — this route renders the
 * same component until the key field is factored into its own page.
 */
export function SettingsMetadataSource() {
  return <SettingsGeneral />
}
