import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SystemTasks hosts /system/tasks — scheduled tasks with last/next run
 * times and manual trigger buttons, plus the current queued-commands list.
 * The scheduler already runs RSS sync, health check, housekeeping, backup,
 * etc.; the UI lists them and lets the user trigger them on demand.
 */
export function SystemTasks() {
  return (
    <PagePlaceholder
      title="Tasks"
      description="Scheduled tasks and queued commands. RssSync, RefreshMonitoredDownloads, HealthCheck, Housekeeping, Backup, and MessagingCleanup run on their configured intervals. UI listing plus trigger buttons are pending."
    />
  )
}
