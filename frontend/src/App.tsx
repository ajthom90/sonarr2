import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './layout/AppShell'
import { SeriesIndex } from './pages/SeriesIndex'
import { Calendar } from './pages/Calendar'
import { Activity } from './pages/Activity'
import { Wanted } from './pages/Wanted'
import { Settings } from './pages/Settings'
import { System } from './pages/System'
import { NotFound } from './pages/NotFound'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<SeriesIndex />} />
            <Route path="series" element={<SeriesIndex />} />
            <Route path="calendar" element={<Calendar />} />
            <Route path="activity" element={<Activity />} />
            <Route path="wanted" element={<Wanted />} />
            <Route path="settings" element={<Settings />} />
            <Route path="system" element={<System />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
