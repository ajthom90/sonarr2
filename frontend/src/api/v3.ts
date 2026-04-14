import { apiFetchRaw, ApiError } from './client'

async function v3<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await apiFetchRaw(`/api/v3${path}`, init)
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as Record<string, unknown>
    const message = (body.message ?? body.detail ?? res.statusText) as string
    throw new ApiError(res.status, message, body)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const apiV3 = {
  get: <T>(path: string) => v3<T>(path),
  post: <T>(path: string, body?: unknown) =>
    v3<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body: unknown) =>
    v3<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (path: string) => v3<void>(path, { method: 'DELETE' }),
}
