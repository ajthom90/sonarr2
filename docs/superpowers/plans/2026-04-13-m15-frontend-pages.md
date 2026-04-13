# Milestone 15 — Frontend Pages

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the placeholder pages with real, functional UI that talks to the v6 API. After M15, users can browse their series library, view episodes, check the calendar, see activity history, manage settings (indexers, download clients, profiles), and monitor system status — all from the web UI.

**Architecture:** Each page uses TanStack Query hooks to fetch data from the v6 API. CSS Modules for styling. No global state store beyond React Query's cache — URL params drive filtering/pagination.

---

## Task 1 — Series list page

Replace the `SeriesIndex` placeholder with a real series list.

**src/pages/SeriesIndex.tsx:**
- Fetch `GET /api/v6/series` via TanStack Query
- Display a table/grid of series with: title, network, status (badge), episode progress (X/Y), size on disk, monitored toggle
- Sort by title (default)
- Loading spinner while fetching
- Empty state: "No series added yet"

**src/api/hooks.ts** (shared hooks file):
```ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type { Series, Page } from './types'

export function useSeriesList() {
  return useQuery({
    queryKey: ['series'],
    queryFn: () => api.get<Page<Series>>('/series'),
  })
}

export function useSystemStatus() {
  return useQuery({
    queryKey: ['system', 'status'],
    queryFn: () => api.get<SystemStatus>('/system/status'),
  })
}
```

**Styling:** CSS Module with a table layout, status badges (green=continuing, red=ended), progress bar for episode completion.

Commit: `feat(frontend): add series list page with API data`

---

## Task 2 — Series detail + episodes page

Add a series detail view at `/series/:id` showing series info and an episode table grouped by season.

**src/pages/SeriesDetail.tsx:**
- Fetch series by ID: `GET /api/v6/series/{id}`
- Fetch episodes: `GET /api/v6/episode?seriesId={id}`
- Show: title, year, status, path, quality profile, overview
- Episodes table grouped by season: episode number, title, air date, file status (has file / missing), monitored checkbox
- Monitored toggle via `PUT /api/v6/episode/{id}`

**Add route** in App.tsx: `<Route path="series/:id" element={<SeriesDetail />} />`

**Hooks:**
```ts
export function useSeries(id: number) {
  return useQuery({
    queryKey: ['series', id],
    queryFn: () => api.get<Series>(`/series/${id}`),
  })
}

export function useEpisodes(seriesId: number) {
  return useQuery({
    queryKey: ['episodes', seriesId],
    queryFn: () => api.get<Page<Episode>>(`/episode?seriesId=${seriesId}`),
  })
}
```

Commit: `feat(frontend): add series detail page with episode list`

---

## Task 3 — Calendar page

Replace the calendar placeholder with an episode airing schedule.

**src/pages/Calendar.tsx:**
- Fetch `GET /api/v6/calendar?start={weekStart}&end={weekEnd}`
- Display episodes grouped by day in a simple list/agenda view (not a full calendar grid — that's complex; a day-by-day list is M15 scope)
- Each entry: series title, episode number, episode title, air time
- Navigate forward/backward by week

Commit: `feat(frontend): add calendar page with weekly episode schedule`

---

## Task 4 — Activity page (history + queue)

**src/pages/Activity.tsx:**
- Two tabs: Queue and History
- **Queue tab:** Fetch `GET /api/v6/command` and show active/recent commands (command name, status, duration)
- **History tab:** Fetch `GET /api/v6/history` with cursor pagination. Show: date, series, episode, event type (grabbed/imported/failed), source title
- Simple tab switching via URL hash or local state

Commit: `feat(frontend): add activity page with queue and history tabs`

---

## Task 5 — Wanted + System pages

**src/pages/Wanted.tsx:**
- Fetch `GET /api/v6/wanted/missing` with pagination
- Table: series, season, episode, title, air date
- Empty state: "No missing episodes"

**src/pages/System.tsx:**
- Fetch `GET /api/v6/system/status`
- Display: app version, database type, runtime, start time, OS info
- Show scheduled tasks from `GET /api/v6/command` (recent commands as a proxy for task history)
- Health checks from `GET /api/v6/health`

Commit: `feat(frontend): add wanted and system status pages`

---

## Task 6 — Settings page (indexers + download clients + profiles)

**src/pages/Settings.tsx:**
- Sub-routes or tabs: Indexers, Download Clients, Quality Profiles
- **Indexers tab:** List configured indexers from `GET /api/v6/indexer`. Add button opens a form (implementation name + settings JSON). Delete button.
- **Download Clients tab:** Same pattern from `GET /api/v6/downloadclient`.
- **Quality Profiles tab:** List profiles from `GET /api/v6/qualityprofile` with name and quality count.

For M15, forms use simple JSON textarea for settings (the dynamic schema-driven forms are a later polish). CRUD operations via the API.

Commit: `feat(frontend): add settings page with indexer, download client, and profile management`

---

## Task 7 — Connection status indicator + README + push

**TopBar enhancement:** show a connection status dot in the top bar that turns green when `/ping` succeeds and red when it fails. Poll every 30 seconds.

```ts
export function useConnectionStatus() {
  return useQuery({
    queryKey: ['ping'],
    queryFn: async () => {
      const res = await fetch('/ping')
      return res.ok
    },
    refetchInterval: 30_000,
  })
}
```

**Update README:** bump to M15, add all implemented pages to the list.

**Final:** `npm run build`, `make build`, smoke test the binary, push, CI.

---

## Done

After Task 7, the web UI is functional: users can browse series, view episodes, check the calendar, see activity history, manage indexers/download clients/profiles, and monitor system health. The UI talks to the v6 API with cursor pagination and live connection status. This is the first milestone where sonarr2 looks like a real application.
