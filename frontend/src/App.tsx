import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthGuard } from './auth/AuthGuard'
import { AppShell } from './layout/AppShell'
import { Login } from './pages/Login'
import { Setup } from './pages/Setup'
import { SeriesIndex } from './pages/SeriesIndex'
import { SeriesDetail } from './pages/SeriesDetail'
import { Calendar } from './pages/Calendar'
import { Activity } from './pages/Activity'
import { Wanted } from './pages/Wanted'
import { Settings } from './pages/Settings'
import { SettingsGeneral } from './pages/SettingsGeneral'
import { System } from './pages/System'
import { AddSeries } from './pages/AddSeries'
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
              <Route index element={<SeriesIndex />} />
              <Route path="series/:id" element={<SeriesDetail />} />
              <Route path="series" element={<SeriesIndex />} />
              <Route path="add/new" element={<AddSeries />} />
              <Route path="calendar" element={<Calendar />} />
              <Route path="activity" element={<Activity />} />
              <Route path="wanted" element={<Wanted />} />
              <Route path="settings" element={<Settings />} />
              <Route path="settings/general" element={<SettingsGeneral />} />
              <Route path="system" element={<System />} />
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </AuthGuard>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
