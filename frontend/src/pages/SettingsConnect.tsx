import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SettingsConnect hosts /settings/connect (notifications). Backend CRUD is
 * live at /api/v3/notification. Sonarr labels this page "Connect"; the full
 * 25-provider list plus 13 per-provider event triggers are pending.
 */
export function SettingsConnect() {
  return (
    <PagePlaceholder
      title="Connect"
      description="Notification integrations: Discord, Slack, Telegram, Email, Webhook, Pushover, Gotify, Custom Script, and more. Event triggers (OnGrab / OnImport / OnUpgrade / OnHealth / etc.) are configured per-provider."
    />
  )
}
