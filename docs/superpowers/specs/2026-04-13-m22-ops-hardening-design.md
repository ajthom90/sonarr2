# M22 — Ops Hardening

## Overview

Add operational security features: URLBase routing for reverse proxy support, security headers, CORS, per-IP rate limiting on API endpoints, and an authentication health check.

## Features

### 1. URLBase Routing

Wire `cfg.HTTP.URLBase` (already parsed from config/env) into the chi router. When URLBase is `/sonarr`, all routes shift:
- `/api/v3/series` → `/sonarr/api/v3/series`
- `/api/v6/series` → `/sonarr/api/v6/series`
- `/ping` → `/sonarr/ping`
- SPA fallback → `/sonarr/*`

Implementation: wrap the entire router in `r.Route(urlBase, ...)` when URLBase is non-empty. Update the `system/status` endpoint to return the actual URLBase value.

### 2. Security Headers Middleware

Add a middleware that sets standard security headers on every response:
- `X-Frame-Options: SAMEORIGIN`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer-when-downgrade`
- `X-XSS-Protection: 0` (modern browsers don't need it, but signals awareness)

### 3. CORS Middleware

Permissive CORS for API key-authenticated requests (standard for PVR apps):
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, X-Api-Key, Authorization`
- Handle OPTIONS preflight requests

This matches Sonarr's behavior — API key auth means CORS restrictions add no security value.

### 4. Per-IP Rate Limiting

Token bucket rate limiter per client IP on `/api/*` routes:
- Default: 100 requests/second burst, 30 sustained
- Uses `golang.org/x/time/rate` (already a dependency)
- LRU map of IP → limiter (bounded to prevent memory growth)
- Returns HTTP 429 with `Retry-After` header when exceeded
- Configurable via `SONARR2_RATE_LIMIT` and `SONARR2_RATE_BURST`

### 5. Auth Health Check

Add `AuthenticationCheck` to the health system:
- Warns if API key is the default/empty
- Warns if auth mode is "none"

## Configuration

| Variable | Default | Description |
|---|---|---|
| `SONARR2_RATE_LIMIT` | `30` | Sustained requests/second per IP |
| `SONARR2_RATE_BURST` | `100` | Burst capacity per IP |

URLBase is already configurable via `SONARR2_URL_BASE`.

## Testing

- **URLBase**: test that routes respond under the prefix and 404 without it
- **Security headers**: test headers are present on responses
- **CORS**: test OPTIONS preflight returns correct headers
- **Rate limiting**: test that requests within limit pass, excess returns 429
- **Auth health check**: test with empty API key → warning

## Out of Scope

- Forms/Basic/External auth modes (API key is sufficient for v1)
- TLS/HTTPS (handled by reverse proxy)
- Per-user rate limiting
- IP allowlists/blocklists
