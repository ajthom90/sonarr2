# Milestone 14 — Frontend Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Scaffold the React + TypeScript + Vite frontend. After M14, running `npm run dev` serves a working app shell with sidebar navigation, dark theme, and the dev proxy to the Go backend. The frontend talks to the v6 API and shows a placeholder page for each route. M15 fills in the actual page content.

**Architecture:** `frontend/` is a standalone Vite + React 18 + TypeScript project. It builds to `frontend/dist/` which the Go binary embeds via `//go:embed`. Dev mode uses Vite's proxy to forward `/api/*` and `/signalr/*` to the Go backend on `:8989`.

---

## Layout

```
frontend/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── src/
│   ├── main.tsx              # Entry point
│   ├── App.tsx               # Router + providers
│   ├── api/
│   │   ├── client.ts         # fetch wrapper with API key header
│   │   └── types.ts          # TypeScript types matching v6 API
│   ├── components/
│   │   ├── Button.tsx
│   │   ├── PageHeader.tsx
│   │   └── Spinner.tsx
│   ├── layout/
│   │   ├── AppShell.tsx      # Sidebar + content area
│   │   ├── Sidebar.tsx       # Navigation links
│   │   └── TopBar.tsx        # Title + status indicator
│   ├── pages/
│   │   ├── SeriesIndex.tsx   # Placeholder
│   │   ├── Calendar.tsx      # Placeholder
│   │   ├── Activity.tsx      # Placeholder
│   │   ├── Wanted.tsx        # Placeholder
│   │   ├── Settings.tsx      # Placeholder
│   │   ├── System.tsx        # Placeholder
│   │   └── NotFound.tsx
│   ├── providers/
│   │   └── QueryProvider.tsx # TanStack Query client
│   ├── styles/
│   │   ├── tokens.css        # CSS custom properties (dark theme)
│   │   ├── reset.css
│   │   └── globals.css
│   └── hooks/
│       └── useApiKey.ts      # Read API key from local storage
└── public/
    └── favicon.svg
```

---

## Task 1 — Initialize frontend project

Create the Vite + React + TypeScript project with all dependencies.

### Steps

```bash
cd /Users/ajthom90/projects/sonarr2
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install @tanstack/react-query react-router-dom
npm install -D @types/react @types/react-dom
```

Then customize:

**vite.config.ts:**
```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8989',
      '/signalr': { target: 'http://localhost:8989', ws: true },
      '/ping': 'http://localhost:8989',
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
```

**tsconfig.json:** ensure `strict: true`, `noUncheckedIndexedAccess: true`.

**package.json scripts:**
```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "tsc --noEmit"
  }
}
```

Clean up the Vite template files (remove default App.css, logo, counter component).

Commit: `feat(frontend): initialize Vite + React + TypeScript project`

---

## Task 2 — Design tokens + CSS reset + dark theme

Create the styling foundation.

**src/styles/tokens.css:**
```css
:root {
  --color-bg: #1a1a2e;
  --color-surface: #16213e;
  --color-surface-raised: #0f3460;
  --color-border: #1a1a4e;
  --color-text: #e0e0e0;
  --color-text-secondary: #999;
  --color-accent: #e94560;
  --color-success: #27ae60;
  --color-warning: #f39c12;
  --color-danger: #e74c3c;
  --color-monitored: #27ae60;
  --color-unmonitored: #666;

  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;

  --radius-sm: 4px;
  --radius-md: 8px;

  --font-sans: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'JetBrains Mono', Menlo, Consolas, monospace;

  --sidebar-width: 220px;
}
```

**src/styles/reset.css:** basic CSS reset (box-sizing, margin 0, etc.)

**src/styles/globals.css:**
```css
@import './reset.css';
@import './tokens.css';

body {
  font-family: var(--font-sans);
  background: var(--color-bg);
  color: var(--color-text);
  min-height: 100vh;
}

#root {
  min-height: 100vh;
}
```

Commit: `feat(frontend): add dark theme design tokens and CSS foundation`

---

## Task 3 — App shell: sidebar + routing

Create the layout components and React Router setup.

**src/layout/Sidebar.tsx:**
Vertical sidebar with links: Series, Calendar, Activity, Wanted, Settings, System. Uses NavLink for active state styling.

**src/layout/TopBar.tsx:**
Horizontal bar with app title "sonarr2" and a connection status indicator.

**src/layout/AppShell.tsx:**
Flexbox layout: Sidebar (fixed width) + content area. Renders `<Outlet />` for the routed page.

**src/App.tsx:**
```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './layout/AppShell'
import { SeriesIndex } from './pages/SeriesIndex'
import { Calendar } from './pages/Calendar'
// ... other pages

const queryClient = new QueryClient()

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
```

**Placeholder pages:** each is a simple component:
```tsx
export function SeriesIndex() {
  return <div><h1>Series</h1><p>Coming in M15</p></div>
}
```

Commit: `feat(frontend): add app shell with sidebar navigation and routing`

---

## Task 4 — API client + types

Create the typed fetch wrapper and TypeScript types matching the v6 API.

**src/api/client.ts:**
```ts
const API_BASE = '/api/v6'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const apiKey = localStorage.getItem('sonarr2_api_key') || ''
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-Api-Key': apiKey,
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new ApiError(res.status, body.detail || body.message || res.statusText)
  }
  return res.json()
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),
  post: <T>(path: string, body: unknown) => apiFetch<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) => apiFetch<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (path: string) => apiFetch<void>(path, { method: 'DELETE' }),
}
```

**src/api/types.ts:**
```ts
export interface Series {
  id: number
  title: string
  sortTitle: string
  status: string
  seriesType: string
  tvdbId: number
  path: string
  monitored: boolean
  year: number
  statistics?: SeriesStatistics
}

export interface SeriesStatistics {
  episodeCount: number
  episodeFileCount: number
  sizeOnDisk: number
  percentOfEpisodes: number
}

export interface Page<T> {
  data: T[]
  pagination: { limit: number; nextCursor?: string; hasMore: boolean }
}

// ... Episode, EpisodeFile, QualityProfile, Command, HistoryEntry types
```

Commit: `feat(frontend): add API client and TypeScript types`

---

## Task 5 — Embed frontend in Go binary + README + push

1. Update the Go build to embed `frontend/dist/`:
   - Create `internal/api/frontend.go` with `//go:embed` for the built frontend
   - Update `server.go` to serve static files from the embedded FS
   - SPA fallback: unknown paths return `index.html`

2. Update `Makefile`: add a `frontend` target that runs `cd frontend && npm ci && npm run build`

3. Update `docker/Dockerfile`: add a frontend build stage before the Go build

4. Update README: bump to M14, add frontend foundation to implemented list

5. tidy, lint, test, build (including frontend), push, CI

**Note:** The `//go:embed` only works if `frontend/dist/` exists at build time. For `go test` without building the frontend first, use a build tag or conditional embed. Simplest: embed with a fallback that returns 404 if the dir doesn't exist.

---

## Done

After Task 5, `npm run dev` serves a dark-themed app shell with sidebar navigation at localhost:5173, proxying API calls to the Go backend. `make build` produces a single binary with the frontend embedded. M15 fills in the actual page content (series list, calendar, activity, etc.).
