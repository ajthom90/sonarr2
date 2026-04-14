import { Activity } from './Activity'

/**
 * ActivityHistory is the /activity/history route. Reuses the Activity page;
 * the page component may later be split into standalone components per tab.
 */
export function ActivityHistory() {
  return <Activity />
}
