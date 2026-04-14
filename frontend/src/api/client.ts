const API_BASE = '/api/v6'

export class ApiError extends Error {
  status: number
  /**
   * details carries the full decoded JSON body from the server's error
   * response, minus the already-extracted `message`/`detail` field. Callers
   * can reach into it for endpoint-specific hints like `fixPath` (from
   * /api/v3/libraryimport/scan) or `affectedSeries` (from DELETE
   * /api/v3/rootfolder/{id}). Empty when the body isn't valid JSON.
   */
  details: Record<string, unknown>
  constructor(status: number, message: string, details: Record<string, unknown> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.details = details
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
    const body = (await res.json().catch(() => ({}))) as Record<string, unknown>
    const message = (body.detail ?? body.message ?? res.statusText) as string
    throw new ApiError(res.status, message, body)
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
