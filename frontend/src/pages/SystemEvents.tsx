import { PagePlaceholder } from '../components/PagePlaceholder'

/**
 * SystemEvents hosts /system/events — the in-database event log filterable
 * by level, component, and search. Backing store, filters, and UI are
 * pending (Sonarr feature: `LogsTableConnector`).
 */
export function SystemEvents() {
  return (
    <PagePlaceholder
      title="Events"
      description="Filterable in-DB event log (Trace/Debug/Info/Warn/Error/Fatal) with component and search filters. Backend log store and ingestion hook are pending."
    />
  )
}
