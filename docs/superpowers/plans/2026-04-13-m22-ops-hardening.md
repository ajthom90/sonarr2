# M22 — Ops Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add URLBase routing, security headers, CORS, per-IP rate limiting, and auth health check.

**Architecture:** New middleware functions in `internal/api/` for security headers, CORS, and rate limiting. URLBase wiring in the router builder. Auth health check added to the existing health package.

**Tech Stack:** Go stdlib + `golang.org/x/time/rate` (already a dep)

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `internal/api/middleware.go` | Security headers, CORS, rate limiting middleware |
| Create | `internal/api/middleware_test.go` | Middleware tests |
| Modify | `internal/api/server.go` | Wire URLBase routing + new middleware, add config to Deps |
| Create | `internal/health/auth.go` | Auth health check |
| Create | `internal/health/auth_test.go` | Auth health check tests |
| Modify | `internal/app/app.go` | Wire new middleware config + auth health check |
| Modify | `README.md` | Update for M22 |

---

### Task 1: Security Middleware

**Files:**
- Create: `internal/api/middleware.go`
- Create: `internal/api/middleware_test.go`

Three middleware functions:

**securityHeaders** — sets standard headers on every response:
```go
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}
```

**corsMiddleware** — handles CORS preflight and response headers:
```go
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Api-Key, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

**ipRateLimiter** — per-IP token bucket with bounded LRU:
```go
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
	maxIPs   int // cap map size (default 10000)
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(rps float64, burst int) *ipRateLimiter

func (l *ipRateLimiter) Middleware(next http.Handler) http.Handler
// extracts client IP from X-Forwarded-For or RemoteAddr
// if limiter.Allow() returns false, respond 429
```

**Tests:**
- `TestSecurityHeaders` — verify all 4 headers present
- `TestCORSPreflight` — OPTIONS returns 204 with correct headers
- `TestCORSNormalRequest` — GET includes CORS headers
- `TestRateLimiterAllows` — requests within limit pass
- `TestRateLimiterBlocks` — excess requests get 429

### Task 2: URLBase Routing + Wire Middleware

**Files:**
- Modify: `internal/api/server.go`

Add `URLBase string` and rate limit config to `Deps`:
```go
URLBase   string
RateLimit float64
RateBurst int
```

In `HandlerWithDeps`, add middleware after Recoverer:
```go
r.Use(securityHeaders)
r.Use(corsMiddleware)

if deps.RateLimit > 0 {
	rl := newIPRateLimiter(deps.RateLimit, deps.RateBurst)
	r.Use(rl.Middleware)
}
```

For URLBase, wrap the entire existing router content inside `r.Route(deps.URLBase, ...)` when URLBase is non-empty. When empty, keep current behavior.

### Task 3: Auth Health Check

**Files:**
- Create: `internal/health/auth.go`
- Create: `internal/health/auth_test.go`

```go
type AuthCheck struct {
	apiKey   string
	authMode string
}

func NewAuthCheck(apiKey, authMode string) *AuthCheck
func (c *AuthCheck) Name() string { return "AuthenticationCheck" }
func (c *AuthCheck) Check(_ context.Context) []Result
// warns if authMode is "none"
```

Tests with empty key and "none" auth mode.

### Task 4: Wire in app.go + Config + README

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/config/config.go` + test
- Modify: `README.md`

Add rate limit config fields (reuse TVDB rate limit pattern). Add `URLBase`, `RateLimit`, `RateBurst` to api.Deps. Add AuthCheck to the Checker. Update README.
