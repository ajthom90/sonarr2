import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsUI hosts /settings/ui: theme (light/dark/auto), UI language,
 * first day of week, short/long date format, time format, relative
 * dates, color-impaired mode. Persistence endpoint /api/v3/config/ui
 * pending.
 */
export function SettingsUI() {
  return (
    <PagePlaceholder
      title="UI"
      description="Theme, language, and date/time display preferences. The config/ui endpoint and per-user persistence are pending."
    />
  )
}
