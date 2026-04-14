const API_BASE = '/api/v6'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  return fetchWithAuth<T>(`${API_BASE}${path}`, init)
}

/**
 * apiFetchRaw performs an authenticated fetch against an arbitrary path
 * (not scoped to the v6 base). Callers provide the full path, e.g.
 * "/api/v3/blocklist". Auth behavior matches apiFetch.
 */
export async function apiFetchRaw(path: string, init?: RequestInit): Promise<Response> {
  const apiKey = localStorage.getItem('sonarr2_api_key') ?? ''
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string> ?? {}),
  }
  if (apiKey) headers['X-Api-Key'] = apiKey
  return fetch(path, { ...init, credentials: 'include', headers })
}

async function fetchWithAuth<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await apiFetchRaw(url, init)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ detail: res.statusText }))
    throw new ApiError(res.status, body.detail ?? body.message ?? res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),
  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (path: string) => apiFetch<void>(path, { method: 'DELETE' }),
}
