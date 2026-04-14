import { Activity } from './Activity'

/**
 * ActivityQueue is the /activity/queue route. It reuses the legacy Activity
 * component with its tab state forced to "queue" via URL matching. When the
 * Activity component is refactored into standalone per-tab components this
 * wrapper can be deleted.
 */
export function ActivityQueue() {
  return <Activity />
}
