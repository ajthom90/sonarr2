# M17 — TVDB Caching & Rate Limiting

## Overview

Add an in-process caching layer and rate-limited HTTP transport for TVDB API calls. This reduces redundant API requests, prevents quota exhaustion, and improves responsiveness for users with large libraries — all without requiring a separate service.

The design spec originally called for a standalone `sonarr2-metadata-proxy` companion service. This milestone implements the in-process variant instead (simpler ops, fits the single-binary philosophy). An optional external proxy can be added later if multi-instance shared caching becomes a requirement.

## Architecture

### CachedMetadataSource Decorator

A new package `internal/providers/metadatasource/cached/` provides a `CachedMetadataSource` struct that implements the existing `MetadataSource` interface and wraps any underlying source.

```
CachedMetadataSource {
    inner     MetadataSource
    cache     map[string]cacheEntry
    mu        sync.RWMutex
    opts      Options
}

cacheEntry {
    value     any
    expiresAt time.Time
}
```

#### Cache keys and TTLs

| Method | Cache key pattern | Default TTL | Rationale |
|---|---|---|---|
| SearchSeries(query) | `search:{query}` | 1 hour | Search results change as new series are added |
| GetSeries(tvdbID) | `series:{id}` | 24 hours | Series metadata is stable |
| GetEpisodes(tvdbID) | `episodes:{id}` | 6 hours | New episodes get announced periodically |

#### Cache behavior

- **Read path**: Check cache under RLock. If entry exists and not expired, return it. Otherwise, call inner source, store result under write lock, return.
- **Expiry sweeping**: A background goroutine runs every 10 minutes, acquires write lock, removes all expired entries. Goroutine stops when a context is cancelled (app shutdown).
- **Invalidation**: `Invalidate(tvdbID int64)` removes `series:{id}` and `episodes:{id}` entries. Called by `RefreshSeriesMetadata` handler before fetching, so user-initiated refreshes always hit TVDB. Search cache entries are not invalidated (they expire naturally).
- **No persistence**: Cache is in-memory only. Lost on restart, fills naturally as users browse and refresh. Acceptable for the homelab use case where restarts are infrequent.

### Rate-Limited HTTP Transport

A new `internal/providers/metadatasource/tvdb/ratelimit.go` file provides a `RateLimitedTransport` that implements `http.RoundTripper`:

```
RateLimitedTransport {
    inner      http.RoundTripper
    limiter    *rate.Limiter    // golang.org/x/time/rate
    maxRetries int
}
```

#### Token bucket

Uses `golang.org/x/time/rate.NewLimiter(rate.Every(200ms), 10)` by default — 5 requests/second steady state with a burst of 10. Configurable via settings.

#### 429 handling

When the upstream TVDB API returns HTTP 429:

1. Read `Retry-After` header. If present and numeric, sleep that many seconds.
2. Otherwise, exponential backoff: 1s, 2s, 4s.
3. Maximum 3 retries before returning the 429 error to the caller.
4. Each retry re-checks the token bucket limiter before sending.

#### Wiring

`tvdb.New()` already accepts an `*http.Client` parameter. The app composition root constructs:

```go
transport := NewRateLimitedTransport(http.DefaultTransport, rateLimitOpts)
httpClient := &http.Client{Transport: transport}
tvdbSource := tvdb.New(settings, httpClient)
```

Zero changes to the existing `tvdb` package API.

## Configuration

New environment variables with defaults:

| Variable | Default | Description |
|---|---|---|
| `SONARR2_TVDB_CACHE_SERIES_TTL` | `24h` | TTL for series metadata cache entries |
| `SONARR2_TVDB_CACHE_EPISODES_TTL` | `6h` | TTL for episode list cache entries |
| `SONARR2_TVDB_CACHE_SEARCH_TTL` | `1h` | TTL for search result cache entries |
| `SONARR2_TVDB_RATE_LIMIT` | `5` | Max requests per second to TVDB API |
| `SONARR2_TVDB_RATE_BURST` | `10` | Burst capacity for rate limiter |

These are parsed in the config package and passed through to the cache and transport constructors.

## App Wiring

In `internal/app/app.go`, the metadata source construction changes from:

```go
tvdbSource := tvdb.New(tvdb.Settings{ApiKey: key}, nil)
```

to:

```go
// Rate-limited HTTP client
transport := tvdb.NewRateLimitedTransport(http.DefaultTransport, tvdb.RateLimitOptions{
    RequestsPerSecond: cfg.TVDBRateLimit,
    Burst:             cfg.TVDBRateBurst,
    MaxRetries:        3,
})
httpClient := &http.Client{Transport: transport}

// TVDB source with rate limiting
tvdbSource := tvdb.New(tvdb.Settings{ApiKey: key}, httpClient)

// Wrap with cache decorator
cachedSource := cached.New(tvdbSource, cached.Options{
    SeriesTTL:   cfg.TVDBCacheSeriesTTL,
    EpisodesTTL: cfg.TVDBCacheEpisodesTTL,
    SearchTTL:   cfg.TVDBCacheSearchTTL,
})

// Pass cached source to handlers
refreshHandler := handlers.NewRefreshSeriesHandler(cachedSource, lib)
```

The `RefreshSeriesMetadata` handler gains an optional `Invalidator` interface check:

```go
type Invalidator interface {
    Invalidate(tvdbID int64)
}
```

If the metadata source implements `Invalidator`, the handler calls `Invalidate(tvdbID)` before fetching. This ensures user-initiated refreshes bypass the cache without coupling the handler to the cache implementation.

## Testing

### Cache tests (`cached/cached_test.go`)

- **Hit**: Second call for same key returns cached value, inner source called once
- **Miss**: First call passes through to inner source
- **Expiry**: Entry expires after TTL, next call hits inner source again
- **Invalidate**: After invalidation, next call hits inner source
- **Concurrent access**: Multiple goroutines reading/writing simultaneously (race detector)
- **Sweep**: Expired entries are cleaned up by background goroutine

### Rate limiter tests (`tvdb/ratelimit_test.go`)

- **Within limit**: Requests under rate pass through immediately
- **Burst**: Burst capacity allows short spikes
- **429 retry**: Mock server returns 429, transport retries with backoff, succeeds on retry
- **429 with Retry-After**: Respects the header value
- **Max retries exceeded**: Returns error after 3 failed attempts
- **Rate limit wait**: Requests beyond burst block until tokens available

### Integration test

- Decorator wrapping a mock `MetadataSource`, full cache-miss → cache-hit → invalidate → cache-miss cycle

## Dependencies

- `golang.org/x/time/rate` — standard Go rate limiter (token bucket). May already be a transitive dependency; if not, it's a single `go get`.

## Out of Scope

- External proxy service (deferred to post-v1 if multi-instance caching is needed)
- Redis or persistent cache storage
- TMDb/TVMaze metadata sources
- Cache size limits (TVDB entries are small; even 10k series < 50MB)
