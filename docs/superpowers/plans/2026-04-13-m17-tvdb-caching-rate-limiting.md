# M17 — TVDB Caching & Rate Limiting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an in-process TTL cache decorator and rate-limited HTTP transport for TVDB API calls, reducing redundant requests and preventing quota exhaustion.

**Architecture:** A `CachedMetadataSource` decorator wraps any `MetadataSource` with per-method TTL caching. A `RateLimitedTransport` implements `http.RoundTripper` with a token bucket rate limiter and 429-aware retry. Both are wired in `app.go` between the TVDB client and command handlers. The `RefreshSeriesMetadata` handler invalidates cache entries before fetching so user-initiated refreshes always hit TVDB.

**Tech Stack:** Go stdlib (`sync.RWMutex`, `time`, `net/http`), `golang.org/x/time/rate` (already a transitive dep)

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `internal/providers/metadatasource/cached/cached.go` | CachedMetadataSource decorator: TTL map, sweep goroutine, Invalidate |
| Create | `internal/providers/metadatasource/cached/cached_test.go` | Cache hit/miss/expiry/invalidate/concurrent tests |
| Create | `internal/providers/metadatasource/tvdb/ratelimit.go` | RateLimitedTransport: token bucket + 429 retry |
| Create | `internal/providers/metadatasource/tvdb/ratelimit_test.go` | Rate limiter and 429 retry tests |
| Modify | `internal/config/config.go` | Add TVDBConfig fields to Config struct |
| Modify | `internal/config/config_test.go` | Test new TVDB config env vars |
| Modify | `internal/commands/handlers/refresh_series.go` | Add Invalidator type assertion before fetch |
| Modify | `internal/commands/handlers/refresh_series_test.go` | Test invalidation call |
| Modify | `internal/app/app.go` | Wire rate-limited transport → tvdb.Client → cached decorator |

---

### Task 1: Rate-Limited HTTP Transport

**Files:**
- Create: `internal/providers/metadatasource/tvdb/ratelimit.go`
- Create: `internal/providers/metadatasource/tvdb/ratelimit_test.go`

- [ ] **Step 1: Write the failing test for basic rate limiting**

Create `internal/providers/metadatasource/tvdb/ratelimit_test.go`:

```go
package tvdb_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/tvdb"
)

func TestRateLimitedTransport_PassesThrough(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := tvdb.NewRateLimitedTransport(srv.Client().Transport, tvdb.RateLimitOptions{
		RequestsPerSecond: 100,
		Burst:             100,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/metadatasource/tvdb/ -run TestRateLimitedTransport_PassesThrough -v`
Expected: FAIL — `NewRateLimitedTransport` not defined

- [ ] **Step 3: Write the RateLimitedTransport implementation**

Create `internal/providers/metadatasource/tvdb/ratelimit.go`:

```go
package tvdb

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitOptions configures the rate limiter.
type RateLimitOptions struct {
	RequestsPerSecond float64 // Steady-state requests per second (default 5).
	Burst             int     // Burst capacity (default 10).
	MaxRetries        int     // Max retries on HTTP 429 (default 3).
}

// RateLimitedTransport wraps an http.RoundTripper with a token-bucket rate
// limiter and automatic retry on HTTP 429 responses.
type RateLimitedTransport struct {
	inner      http.RoundTripper
	limiter    *rate.Limiter
	maxRetries int
}

// NewRateLimitedTransport creates a rate-limited transport. Pass nil for inner
// to use http.DefaultTransport.
func NewRateLimitedTransport(inner http.RoundTripper, opts RateLimitOptions) *RateLimitedTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	rps := opts.RequestsPerSecond
	if rps <= 0 {
		rps = 5
	}
	burst := opts.Burst
	if burst <= 0 {
		burst = 10
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &RateLimitedTransport{
		inner:      inner,
		limiter:    rate.NewLimiter(rate.Limit(rps), burst),
		maxRetries: maxRetries,
	}
}

// RoundTrip implements http.RoundTripper. It waits for a rate limiter token,
// sends the request, and retries with exponential backoff on HTTP 429.
func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	if err := t.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for attempt := 0; resp.StatusCode == http.StatusTooManyRequests && attempt < t.maxRetries; attempt++ {
		resp.Body.Close()

		delay := backoffDelay(resp, attempt)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if err := t.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait on retry: %w", err)
		}

		resp, err = t.inner.RoundTrip(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// backoffDelay returns the delay before retrying a 429 response. If the
// response includes a Retry-After header with a numeric value (seconds), that
// is used. Otherwise exponential backoff: 1s, 2s, 4s.
func backoffDelay(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/metadatasource/tvdb/ -run TestRateLimitedTransport_PassesThrough -v`
Expected: PASS

- [ ] **Step 5: Write the 429 retry test**

Add to `internal/providers/metadatasource/tvdb/ratelimit_test.go`:

```go
func TestRateLimitedTransport_Retries429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := tvdb.NewRateLimitedTransport(srv.Client().Transport, tvdb.RateLimitOptions{
		RequestsPerSecond: 1000,
		Burst:             1000,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	// Initial call + 2 retries = 3 total calls to the server
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestRateLimitedTransport_MaxRetriesExceeded(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	transport := tvdb.NewRateLimitedTransport(srv.Client().Transport, tvdb.RateLimitOptions{
		RequestsPerSecond: 1000,
		Burst:             1000,
		MaxRetries:        2,
	})
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	// Should return the 429 after exhausting retries.
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", resp.StatusCode)
	}
	// Initial call + 2 retries = 3 total
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestRateLimitedTransport_RespectsRetryAfterHeader(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := tvdb.NewRateLimitedTransport(srv.Client().Transport, tvdb.RateLimitOptions{
		RequestsPerSecond: 1000,
		Burst:             1000,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	start := time.Now()
	resp, err := client.Get(srv.URL)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	// Should have waited at least ~1 second for the Retry-After header.
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v, expected >= ~1s due to Retry-After header", elapsed)
	}
}
```

- [ ] **Step 6: Run all rate limiter tests**

Run: `go test ./internal/providers/metadatasource/tvdb/ -run TestRateLimitedTransport -v`
Expected: All 4 tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/providers/metadatasource/tvdb/ratelimit.go internal/providers/metadatasource/tvdb/ratelimit_test.go
git commit -m "feat(tvdb): add rate-limited HTTP transport with 429 retry"
```

---

### Task 2: CachedMetadataSource Decorator

**Files:**
- Create: `internal/providers/metadatasource/cached/cached.go`
- Create: `internal/providers/metadatasource/cached/cached_test.go`

- [ ] **Step 1: Write the failing test for cache miss then hit**

Create `internal/providers/metadatasource/cached/cached_test.go`:

```go
package cached_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/cached"
)

// mockSource is a test double that counts calls.
type mockSource struct {
	searchCalls   atomic.Int32
	seriesCalls   atomic.Int32
	episodesCalls atomic.Int32
}

func (m *mockSource) SearchSeries(_ context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	m.searchCalls.Add(1)
	return []metadatasource.SeriesSearchResult{
		{TvdbID: 71663, Title: "The Simpsons: " + query},
	}, nil
}

func (m *mockSource) GetSeries(_ context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	m.seriesCalls.Add(1)
	return metadatasource.SeriesInfo{TvdbID: tvdbID, Title: "The Simpsons"}, nil
}

func (m *mockSource) GetEpisodes(_ context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	m.episodesCalls.Add(1)
	return []metadatasource.EpisodeInfo{
		{TvdbID: 123, SeasonNumber: 1, EpisodeNumber: 1, Title: "Pilot"},
	}, nil
}

func TestCacheMissThenHit(t *testing.T) {
	mock := &mockSource{}
	ctx := context.Background()

	c := cached.New(mock, cached.Options{
		SeriesTTL:   time.Hour,
		EpisodesTTL: time.Hour,
		SearchTTL:   time.Hour,
	})
	defer c.Stop()

	// First call: cache miss — calls inner.
	info1, err := c.GetSeries(ctx, 71663)
	if err != nil {
		t.Fatalf("first GetSeries: %v", err)
	}
	if info1.TvdbID != 71663 {
		t.Errorf("TvdbID = %d, want 71663", info1.TvdbID)
	}
	if mock.seriesCalls.Load() != 1 {
		t.Errorf("seriesCalls = %d, want 1", mock.seriesCalls.Load())
	}

	// Second call: cache hit — does NOT call inner.
	info2, err := c.GetSeries(ctx, 71663)
	if err != nil {
		t.Fatalf("second GetSeries: %v", err)
	}
	if info2.TvdbID != 71663 {
		t.Errorf("TvdbID = %d, want 71663", info2.TvdbID)
	}
	if mock.seriesCalls.Load() != 1 {
		t.Errorf("seriesCalls = %d after cache hit, want 1", mock.seriesCalls.Load())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/metadatasource/cached/ -run TestCacheMissThenHit -v`
Expected: FAIL — package `cached` does not exist

- [ ] **Step 3: Write the CachedMetadataSource implementation**

Create `internal/providers/metadatasource/cached/cached.go`:

```go
// Package cached provides a TTL-caching decorator for metadatasource.MetadataSource.
package cached

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// Options configures cache TTLs.
type Options struct {
	SeriesTTL   time.Duration // TTL for GetSeries results (default 24h).
	EpisodesTTL time.Duration // TTL for GetEpisodes results (default 6h).
	SearchTTL   time.Duration // TTL for SearchSeries results (default 1h).
	SweepEvery  time.Duration // How often to purge expired entries (default 10m).
}

func (o Options) withDefaults() Options {
	if o.SeriesTTL <= 0 {
		o.SeriesTTL = 24 * time.Hour
	}
	if o.EpisodesTTL <= 0 {
		o.EpisodesTTL = 6 * time.Hour
	}
	if o.SearchTTL <= 0 {
		o.SearchTTL = time.Hour
	}
	if o.SweepEvery <= 0 {
		o.SweepEvery = 10 * time.Minute
	}
	return o
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// CachedMetadataSource wraps a MetadataSource with an in-memory TTL cache.
type CachedMetadataSource struct {
	inner metadatasource.MetadataSource
	opts  Options

	mu    sync.RWMutex
	cache map[string]cacheEntry

	stopOnce sync.Once
	stopCh   chan struct{}
}

// New creates a CachedMetadataSource wrapping inner.
func New(inner metadatasource.MetadataSource, opts Options) *CachedMetadataSource {
	opts = opts.withDefaults()
	c := &CachedMetadataSource{
		inner:  inner,
		opts:   opts,
		cache:  make(map[string]cacheEntry),
		stopCh: make(chan struct{}),
	}
	go c.sweepLoop()
	return c
}

// Stop halts the background sweep goroutine. Safe to call multiple times.
func (c *CachedMetadataSource) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// Invalidate removes cached series and episode entries for the given TVDB ID.
// Search entries expire naturally.
func (c *CachedMetadataSource) Invalidate(tvdbID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, fmt.Sprintf("series:%d", tvdbID))
	delete(c.cache, fmt.Sprintf("episodes:%d", tvdbID))
}

// SearchSeries implements metadatasource.MetadataSource with caching.
func (c *CachedMetadataSource) SearchSeries(ctx context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	key := "search:" + query
	if v, ok := c.get(key); ok {
		return v.([]metadatasource.SeriesSearchResult), nil
	}
	result, err := c.inner.SearchSeries(ctx, query)
	if err != nil {
		return nil, err
	}
	c.set(key, result, c.opts.SearchTTL)
	return result, nil
}

// GetSeries implements metadatasource.MetadataSource with caching.
func (c *CachedMetadataSource) GetSeries(ctx context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	key := fmt.Sprintf("series:%d", tvdbID)
	if v, ok := c.get(key); ok {
		return v.(metadatasource.SeriesInfo), nil
	}
	result, err := c.inner.GetSeries(ctx, tvdbID)
	if err != nil {
		return metadatasource.SeriesInfo{}, err
	}
	c.set(key, result, c.opts.SeriesTTL)
	return result, nil
}

// GetEpisodes implements metadatasource.MetadataSource with caching.
func (c *CachedMetadataSource) GetEpisodes(ctx context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	key := fmt.Sprintf("episodes:%d", tvdbID)
	if v, ok := c.get(key); ok {
		return v.([]metadatasource.EpisodeInfo), nil
	}
	result, err := c.inner.GetEpisodes(ctx, tvdbID)
	if err != nil {
		return nil, err
	}
	c.set(key, result, c.opts.EpisodesTTL)
	return result, nil
}

func (c *CachedMetadataSource) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *CachedMetadataSource) set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *CachedMetadataSource) sweepLoop() {
	ticker := time.NewTicker(c.opts.SweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.sweep()
		}
	}
}

func (c *CachedMetadataSource) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, entry := range c.cache {
		if now.After(entry.expiresAt) {
			delete(c.cache, k)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/metadatasource/cached/ -run TestCacheMissThenHit -v`
Expected: PASS

- [ ] **Step 5: Write remaining cache tests**

Add to `internal/providers/metadatasource/cached/cached_test.go`:

```go
func TestCacheExpiry(t *testing.T) {
	mock := &mockSource{}
	ctx := context.Background()

	c := cached.New(mock, cached.Options{
		SeriesTTL:   50 * time.Millisecond,
		EpisodesTTL: time.Hour,
		SearchTTL:   time.Hour,
	})
	defer c.Stop()

	// First call populates cache.
	if _, err := c.GetSeries(ctx, 71663); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if mock.seriesCalls.Load() != 1 {
		t.Fatalf("seriesCalls = %d, want 1", mock.seriesCalls.Load())
	}

	// Wait for TTL to expire.
	time.Sleep(80 * time.Millisecond)

	// Second call should miss cache.
	if _, err := c.GetSeries(ctx, 71663); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if mock.seriesCalls.Load() != 2 {
		t.Errorf("seriesCalls = %d after expiry, want 2", mock.seriesCalls.Load())
	}
}

func TestCacheInvalidate(t *testing.T) {
	mock := &mockSource{}
	ctx := context.Background()

	c := cached.New(mock, cached.Options{
		SeriesTTL:   time.Hour,
		EpisodesTTL: time.Hour,
		SearchTTL:   time.Hour,
	})
	defer c.Stop()

	// Populate both series and episodes caches.
	if _, err := c.GetSeries(ctx, 71663); err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	if _, err := c.GetEpisodes(ctx, 71663); err != nil {
		t.Fatalf("GetEpisodes: %v", err)
	}
	if mock.seriesCalls.Load() != 1 || mock.episodesCalls.Load() != 1 {
		t.Fatalf("expected 1 call each, got series=%d episodes=%d",
			mock.seriesCalls.Load(), mock.episodesCalls.Load())
	}

	// Invalidate.
	c.Invalidate(71663)

	// Next calls should miss cache.
	if _, err := c.GetSeries(ctx, 71663); err != nil {
		t.Fatalf("GetSeries after invalidate: %v", err)
	}
	if _, err := c.GetEpisodes(ctx, 71663); err != nil {
		t.Fatalf("GetEpisodes after invalidate: %v", err)
	}
	if mock.seriesCalls.Load() != 2 {
		t.Errorf("seriesCalls = %d after invalidate, want 2", mock.seriesCalls.Load())
	}
	if mock.episodesCalls.Load() != 2 {
		t.Errorf("episodesCalls = %d after invalidate, want 2", mock.episodesCalls.Load())
	}
}

func TestCacheSearchSeries(t *testing.T) {
	mock := &mockSource{}
	ctx := context.Background()

	c := cached.New(mock, cached.Options{
		SeriesTTL:   time.Hour,
		EpisodesTTL: time.Hour,
		SearchTTL:   time.Hour,
	})
	defer c.Stop()

	// First call: miss.
	results, err := c.SearchSeries(ctx, "simpsons")
	if err != nil {
		t.Fatalf("SearchSeries: %v", err)
	}
	if len(results) != 1 || results[0].TvdbID != 71663 {
		t.Errorf("unexpected results: %v", results)
	}
	if mock.searchCalls.Load() != 1 {
		t.Errorf("searchCalls = %d, want 1", mock.searchCalls.Load())
	}

	// Second call: hit.
	if _, err := c.SearchSeries(ctx, "simpsons"); err != nil {
		t.Fatalf("second SearchSeries: %v", err)
	}
	if mock.searchCalls.Load() != 1 {
		t.Errorf("searchCalls = %d after hit, want 1", mock.searchCalls.Load())
	}

	// Different query: miss.
	if _, err := c.SearchSeries(ctx, "futurama"); err != nil {
		t.Fatalf("SearchSeries futurama: %v", err)
	}
	if mock.searchCalls.Load() != 2 {
		t.Errorf("searchCalls = %d, want 2", mock.searchCalls.Load())
	}
}

func TestCacheConcurrent(t *testing.T) {
	mock := &mockSource{}
	ctx := context.Background()

	c := cached.New(mock, cached.Options{
		SeriesTTL:   time.Hour,
		EpisodesTTL: time.Hour,
		SearchTTL:   time.Hour,
	})
	defer c.Stop()

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = c.GetSeries(ctx, 71663)
			_, _ = c.GetEpisodes(ctx, 71663)
			_, _ = c.SearchSeries(ctx, "simpsons")
			c.Invalidate(71663)
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}
```

- [ ] **Step 6: Run all cache tests (with race detector)**

Run: `go test ./internal/providers/metadatasource/cached/ -race -v`
Expected: All 5 tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/providers/metadatasource/cached/cached.go internal/providers/metadatasource/cached/cached_test.go
git commit -m "feat(metadatasource): add CachedMetadataSource TTL decorator"
```

---

### Task 3: TVDB Config Fields

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test for TVDB env vars**

Add to `internal/config/config_test.go`:

```go
func TestLoadTVDBEnvOverrides(t *testing.T) {
	env := map[string]string{
		"SONARR2_TVDB_API_KEY":            "my-key",
		"SONARR2_TVDB_CACHE_SERIES_TTL":   "48h",
		"SONARR2_TVDB_CACHE_EPISODES_TTL": "12h",
		"SONARR2_TVDB_CACHE_SEARCH_TTL":   "2h",
		"SONARR2_TVDB_RATE_LIMIT":         "10",
		"SONARR2_TVDB_RATE_BURST":         "20",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load(nil, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TVDB.ApiKey != "my-key" {
		t.Errorf("TVDB.ApiKey = %q, want my-key", cfg.TVDB.ApiKey)
	}
	if cfg.TVDB.CacheSeriesTTL != 48*time.Hour {
		t.Errorf("CacheSeriesTTL = %v, want 48h", cfg.TVDB.CacheSeriesTTL)
	}
	if cfg.TVDB.CacheEpisodesTTL != 12*time.Hour {
		t.Errorf("CacheEpisodesTTL = %v, want 12h", cfg.TVDB.CacheEpisodesTTL)
	}
	if cfg.TVDB.CacheSearchTTL != 2*time.Hour {
		t.Errorf("CacheSearchTTL = %v, want 2h", cfg.TVDB.CacheSearchTTL)
	}
	if cfg.TVDB.RateLimit != 10 {
		t.Errorf("RateLimit = %v, want 10", cfg.TVDB.RateLimit)
	}
	if cfg.TVDB.RateBurst != 20 {
		t.Errorf("RateBurst = %d, want 20", cfg.TVDB.RateBurst)
	}
}

func TestTVDBDefaults(t *testing.T) {
	cfg := Default()
	if cfg.TVDB.CacheSeriesTTL != 24*time.Hour {
		t.Errorf("default CacheSeriesTTL = %v, want 24h", cfg.TVDB.CacheSeriesTTL)
	}
	if cfg.TVDB.CacheEpisodesTTL != 6*time.Hour {
		t.Errorf("default CacheEpisodesTTL = %v, want 6h", cfg.TVDB.CacheEpisodesTTL)
	}
	if cfg.TVDB.CacheSearchTTL != time.Hour {
		t.Errorf("default CacheSearchTTL = %v, want 1h", cfg.TVDB.CacheSearchTTL)
	}
	if cfg.TVDB.RateLimit != 5 {
		t.Errorf("default RateLimit = %v, want 5", cfg.TVDB.RateLimit)
	}
	if cfg.TVDB.RateBurst != 10 {
		t.Errorf("default RateBurst = %d, want 10", cfg.TVDB.RateBurst)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run "TestLoadTVDBEnvOverrides|TestTVDBDefaults" -v`
Expected: FAIL — `cfg.TVDB` not defined

- [ ] **Step 3: Add TVDBConfig to config.go**

In `internal/config/config.go`, add the `TVDBConfig` struct and wire it into `Config`, `Default()`, and `Load()`.

Add the struct after `DBConfig`:

```go
// TVDBConfig controls the TVDB metadata source.
type TVDBConfig struct {
	ApiKey           string        `yaml:"api_key"`
	CacheSeriesTTL   time.Duration `yaml:"cache_series_ttl"`
	CacheEpisodesTTL time.Duration `yaml:"cache_episodes_ttl"`
	CacheSearchTTL   time.Duration `yaml:"cache_search_ttl"`
	RateLimit        float64       `yaml:"rate_limit"`
	RateBurst        int           `yaml:"rate_burst"`
}
```

Add `TVDB TVDBConfig \`yaml:"tvdb"\`` to the `Config` struct.

In `Default()`, add:

```go
TVDB: TVDBConfig{
	CacheSeriesTTL:   24 * time.Hour,
	CacheEpisodesTTL: 6 * time.Hour,
	CacheSearchTTL:   time.Hour,
	RateLimit:        5,
	RateBurst:        10,
},
```

In `Load()`, add env var parsing after the existing DB env vars block:

```go
if v := getenv("SONARR2_TVDB_API_KEY"); v != "" {
	cfg.TVDB.ApiKey = v
}
if v := getenv("SONARR2_TVDB_CACHE_SERIES_TTL"); v != "" {
	d, err := time.ParseDuration(v)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_SERIES_TTL must be a duration, got %q: %w", v, err)
	}
	cfg.TVDB.CacheSeriesTTL = d
}
if v := getenv("SONARR2_TVDB_CACHE_EPISODES_TTL"); v != "" {
	d, err := time.ParseDuration(v)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_EPISODES_TTL must be a duration, got %q: %w", v, err)
	}
	cfg.TVDB.CacheEpisodesTTL = d
}
if v := getenv("SONARR2_TVDB_CACHE_SEARCH_TTL"); v != "" {
	d, err := time.ParseDuration(v)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_SEARCH_TTL must be a duration, got %q: %w", v, err)
	}
	cfg.TVDB.CacheSearchTTL = d
}
if v := getenv("SONARR2_TVDB_RATE_LIMIT"); v != "" {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_TVDB_RATE_LIMIT must be a number, got %q: %w", v, err)
	}
	cfg.TVDB.RateLimit = f
}
if v := getenv("SONARR2_TVDB_RATE_BURST"); v != "" {
	n, err := strconv.Atoi(v)
	if err != nil {
		return Config{}, fmt.Errorf("SONARR2_TVDB_RATE_BURST must be an integer, got %q: %w", v, err)
	}
	cfg.TVDB.RateBurst = n
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: All tests PASS (including new TVDB tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add TVDB cache and rate limit configuration"
```

---

### Task 4: Wire Invalidation in RefreshSeriesHandler

**Files:**
- Modify: `internal/commands/handlers/refresh_series.go`
- Modify: `internal/commands/handlers/refresh_series_test.go`

**Context:** The test file is in package `handlers` (internal, not `_test`). It already has helpers: `setupHandlerLib(t)` returns `(*library.Library, *db.SQLitePool)`, `makeCmd(t, seriesID)` builds a Command, and `stubSource` is a test double. We need to add an `invalidatingSource` that also implements `Invalidate`.

- [ ] **Step 1: Write the failing test for invalidation**

Add to `internal/commands/handlers/refresh_series_test.go`:

```go
// invalidatingSource is a stubSource that also tracks Invalidate calls.
type invalidatingSource struct {
	stubSource
	invalidateCalls atomic.Int32
	lastInvalidated atomic.Int64
}

func (s *invalidatingSource) Invalidate(tvdbID int64) {
	s.invalidateCalls.Add(1)
	s.lastInvalidated.Store(tvdbID)
}

func TestRefreshSeriesHandlerCallsInvalidate(t *testing.T) {
	ctx := context.Background()
	lib, _ := setupHandlerLib(t)

	// Create a test series.
	s, err := lib.Series.Create(ctx, library.Series{
		TvdbID:     71663,
		Title:      "The Simpsons",
		Slug:       "the-simpsons",
		Status:     "continuing",
		SeriesType: "standard",
		Path:       "/tv/The Simpsons",
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	source := &invalidatingSource{
		stubSource: stubSource{
			series:   metadatasource.SeriesInfo{TvdbID: 71663, Title: "The Simpsons", Status: "continuing"},
			episodes: nil,
		},
	}

	handler := NewRefreshSeriesHandler(source, lib)
	if err := handler.Handle(ctx, makeCmd(t, s.ID)); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if source.invalidateCalls.Load() != 1 {
		t.Errorf("Invalidate called %d times, want 1", source.invalidateCalls.Load())
	}
	if source.lastInvalidated.Load() != 71663 {
		t.Errorf("Invalidated tvdbID = %d, want 71663", source.lastInvalidated.Load())
	}
}
```

Add `"sync/atomic"` to the imports if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/commands/handlers/ -run TestRefreshSeriesHandler_CallsInvalidate -v`
Expected: FAIL — Invalidate is not called (handler doesn't check for it yet)

- [ ] **Step 3: Add Invalidator check to RefreshSeriesHandler**

In `internal/commands/handlers/refresh_series.go`, add the `Invalidator` interface and the type assertion in `Handle`. Insert right after loading the series (after step 2 "Load series from library"):

```go
// Invalidator is an optional interface for metadata sources that support
// cache invalidation. If the source implements it, Invalidate is called
// before fetching to ensure user-initiated refreshes bypass any cache.
type Invalidator interface {
	Invalidate(tvdbID int64)
}
```

In `Handle`, after loading the series and before calling `GetSeries`, add:

```go
// 2b. Invalidate cache if the source supports it.
if inv, ok := h.source.(Invalidator); ok {
	inv.Invalidate(series.TvdbID)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/commands/handlers/ -run TestRefreshSeriesHandler_CallsInvalidate -v`
Expected: PASS

- [ ] **Step 5: Run full test suite to check for regressions**

Run: `go test ./internal/commands/handlers/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/commands/handlers/refresh_series.go internal/commands/handlers/refresh_series_test.go
git commit -m "feat(handlers): invalidate metadata cache on RefreshSeriesMetadata"
```

---

### Task 5: Wire Everything in app.go

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Update imports in app.go**

Add these imports to `internal/app/app.go`:

```go
"net/http"

"github.com/ajthom90/sonarr2/internal/providers/metadatasource/cached"
```

- [ ] **Step 2: Replace tvdbSource construction**

In `internal/app/app.go`, replace:

```go
// Create the TVDB metadata source. API key is empty by default — users
// configure it via the UI or SONARR2_TVDB_API_KEY env var later. The
// handler will return an error if called without a valid key.
tvdbSource := tvdb.New(tvdb.Settings{ApiKey: ""}, nil)
```

with:

```go
// Create the TVDB metadata source with rate limiting and caching.
// API key comes from config (env var SONARR2_TVDB_API_KEY or YAML).
// The handler will return an error if called without a valid key.
tvdbTransport := tvdb.NewRateLimitedTransport(http.DefaultTransport, tvdb.RateLimitOptions{
	RequestsPerSecond: cfg.TVDB.RateLimit,
	Burst:             cfg.TVDB.RateBurst,
	MaxRetries:        3,
})
tvdbClient := tvdb.New(tvdb.Settings{ApiKey: cfg.TVDB.ApiKey}, &http.Client{Transport: tvdbTransport})
tvdbSource := cached.New(tvdbClient, cached.Options{
	SeriesTTL:   cfg.TVDB.CacheSeriesTTL,
	EpisodesTTL: cfg.TVDB.CacheEpisodesTTL,
	SearchTTL:   cfg.TVDB.CacheSearchTTL,
})
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: Clean build, no errors

- [ ] **Step 4: Run full test suite**

Run: `go test ./... 2>&1`
Expected: All packages PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): wire rate-limited transport and cache decorator for TVDB"
```

---

### Task 6: README Update

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update milestone counter**

Change `**Milestone 16 of 24 complete**` to `**Milestone 17 of 24 complete**` and update the description to mention TVDB caching and rate limiting.

- [ ] **Step 2: Update "What's implemented" section**

After the notification providers bullet, add a new bullet:

```
- **TVDB caching & rate limiting** — in-process TTL cache (24h series, 6h episodes, 1h search) with automatic invalidation on refresh; token-bucket rate limiter (5 req/s) with 429-aware exponential backoff
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README to reflect M17 progress"
```
