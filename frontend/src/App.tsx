import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthGuard } from './auth/AuthGuard'
import { AppShell } from './layout/AppShell'
import { Login } from './pages/Login'
import { Setup } from './pages/Setup'
import { SeriesIndex } from './pages/SeriesIndex'
import { SeriesDetail } from './pages/SeriesDetail'
import { SeriesEditor } from './pages/SeriesEditor'
import { SeasonPass } from './pages/SeasonPass'
import { Calendar } from './pages/Calendar'
import { Activity } from './pages/Activity'
import { ActivityQueue } from './pages/ActivityQueue'
import { ActivityHistory } from './pages/ActivityHistory'
import { ActivityBlocklist } from './pages/ActivityBlocklist'
import { Wanted } from './pages/Wanted'
import { WantedMissing } from './pages/WantedMissing'
import { WantedCutoffUnmet } from './pages/WantedCutoffUnmet'
import { Settings } from './pages/Settings'
import { SettingsGeneral } from './pages/SettingsGeneral'
import { SettingsCustomFormats } from './pages/SettingsCustomFormats'
import { SettingsMediaManagement } from './pages/SettingsMediaManagement'
import { SettingsProfiles } from './pages/SettingsProfiles'
import { SettingsQuality } from './pages/SettingsQuality'
import { SettingsIndexers } from './pages/SettingsIndexers'
import { SettingsDownloadClients } from './pages/SettingsDownloadClients'
import { SettingsImportLists } from './pages/SettingsImportLists'
import { SettingsConnect } from './pages/SettingsConnect'
import { SettingsMetadata } from './pages/SettingsMetadata'
import { SettingsMetadataSource } from './pages/SettingsMetadataSource'
import { SettingsTags } from './pages/SettingsTags'
import { SettingsUI } from './pages/SettingsUI'
import { System } from './pages/System'
import { SystemStatus } from './pages/SystemStatus'
import { SystemTasks } from './pages/SystemTasks'
import { SystemBackup } from './pages/SystemBackup'
import { SystemUpdates } from './pages/SystemUpdates'
import { SystemEvents } from './pages/SystemEvents'
import { SystemLogFiles } from './pages/SystemLogFiles'
import { AddSeries } from './pages/AddSeries'
import { LibraryImport } from './pages/LibraryImport'
import { NotFound } from './pages/NotFound'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthGuard>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/setup" element={<Setup />} />
            <Route element={<AppShell />}>
              {/* Series */}
              <Route index element={<SeriesIndex />} />
              <Route path="series" element={<SeriesIndex />} />
              <Route path="series/:id" element={<SeriesDetail />} />
              <Route path="serieseditor" element={<SeriesEditor />} />
              <Route path="seasonpass" element={<SeasonPass />} />
              <Route path="add/new" element={<AddSeries />} />
              <Route path="add/import" element={<LibraryImport />} />

              {/* Calendar */}
              <Route path="calendar" element={<Calendar />} />

              {/* Activity */}
              <Route path="activity" element={<Activity />} />
              <Route path="activity/queue" element={<ActivityQueue />} />
              <Route path="activity/history" element={<ActivityHistory />} />
              <Route path="activity/blocklist" element={<ActivityBlocklist />} />

              {/* Wanted */}
              <Route path="wanted" element={<Wanted />} />
              <Route path="wanted/missing" element={<WantedMissing />} />
              <Route path="wanted/cutoffunmet" element={<WantedCutoffUnmet />} />

              {/* Settings */}
              <Route path="settings" element={<Settings />} />
              <Route path="settings/mediamanagement" element={<SettingsMediaManagement />} />
              <Route path="settings/profiles" element={<SettingsProfiles />} />
              <Route path="settings/quality" element={<SettingsQuality />} />
              <Route path="settings/customformats" element={<SettingsCustomFormats />} />
              <Route path="settings/indexers" element={<SettingsIndexers />} />
              <Route path="settings/downloadclients" element={<SettingsDownloadClients />} />
              <Route path="settings/importlists" element={<SettingsImportLists />} />
              <Route path="settings/connect" element={<SettingsConnect />} />
              <Route path="settings/metadata" element={<SettingsMetadata />} />
              <Route path="settings/metadatasource" element={<SettingsMetadataSource />} />
              <Route path="settings/tags" element={<SettingsTags />} />
              <Route path="settings/general" element={<SettingsGeneral />} />
              <Route path="settings/ui" element={<SettingsUI />} />

              {/* System */}
              <Route path="system" element={<System />} />
              <Route path="system/status" element={<SystemStatus />} />
              <Route path="system/tasks" element={<SystemTasks />} />
              <Route path="system/backup" element={<SystemBackup />} />
              <Route path="system/updates" element={<SystemUpdates />} />
              <Route path="system/events" element={<SystemEvents />} />
              <Route path="system/logs/files" element={<SystemLogFiles />} />

              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </AuthGuard>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
