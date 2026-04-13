// Package cached provides a TTL-caching decorator for metadatasource.MetadataSource.
// Wrapping any MetadataSource with New reduces external API calls by serving
// repeat requests from an in-memory store until entries expire.
package cached

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// Compile-time check: Source implements MetadataSource.
var _ metadatasource.MetadataSource = (*Source)(nil)

// Options configures TTLs and the background sweep interval.
// Zero values are replaced with sensible defaults by withDefaults.
type Options struct {
	// SeriesTTL is how long a GetSeries result is cached.
	SeriesTTL time.Duration
	// EpisodesTTL is how long a GetEpisodes result is cached.
	EpisodesTTL time.Duration
	// SearchTTL is how long a SearchSeries result is cached.
	SearchTTL time.Duration
	// SweepEvery controls how often the background goroutine purges expired entries.
	SweepEvery time.Duration
}

func (o Options) withDefaults() Options {
	if o.SeriesTTL == 0 {
		o.SeriesTTL = 24 * time.Hour
	}
	if o.EpisodesTTL == 0 {
		o.EpisodesTTL = 6 * time.Hour
	}
	if o.SearchTTL == 0 {
		o.SearchTTL = 1 * time.Hour
	}
	if o.SweepEvery == 0 {
		o.SweepEvery = 10 * time.Minute
	}
	return o
}

// cacheEntry holds a cached value and its expiry time.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Source wraps a MetadataSource and caches results with per-type TTLs.
type Source struct {
	inner    metadatasource.MetadataSource
	opts     Options
	mu       sync.RWMutex
	cache    map[string]cacheEntry
	stopOnce sync.Once
	stopCh   chan struct{}
}

// New creates a Source wrapping inner. A background goroutine
// sweeps expired entries every opts.SweepEvery. Call Stop to terminate it.
func New(inner metadatasource.MetadataSource, opts Options) *Source {
	c := &Source{
		inner:  inner,
		opts:   opts.withDefaults(),
		cache:  make(map[string]cacheEntry),
		stopCh: make(chan struct{}),
	}
	go c.sweepLoop()
	return c
}

// Stop halts the background sweep goroutine. Safe to call multiple times.
func (c *Source) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// Invalidate removes the cached series and episodes entries for tvdbID so the
// next request re-fetches from the inner source.
func (c *Source) Invalidate(tvdbID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, fmt.Sprintf("series:%d", tvdbID))
	delete(c.cache, fmt.Sprintf("episodes:%d", tvdbID))
}

// SearchSeries returns cached results or delegates to the inner source.
func (c *Source) SearchSeries(ctx context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	key := "search:" + query
	if v, ok := c.get(key); ok {
		return v.([]metadatasource.SeriesSearchResult), nil
	}
	results, err := c.inner.SearchSeries(ctx, query)
	if err != nil {
		return nil, err
	}
	c.set(key, results, c.opts.SearchTTL)
	return results, nil
}

// GetSeries returns cached series metadata or delegates to the inner source.
func (c *Source) GetSeries(ctx context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	key := fmt.Sprintf("series:%d", tvdbID)
	if v, ok := c.get(key); ok {
		return v.(metadatasource.SeriesInfo), nil
	}
	info, err := c.inner.GetSeries(ctx, tvdbID)
	if err != nil {
		return metadatasource.SeriesInfo{}, err
	}
	c.set(key, info, c.opts.SeriesTTL)
	return info, nil
}

// GetEpisodes returns cached episode metadata or delegates to the inner source.
func (c *Source) GetEpisodes(ctx context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	key := fmt.Sprintf("episodes:%d", tvdbID)
	if v, ok := c.get(key); ok {
		return v.([]metadatasource.EpisodeInfo), nil
	}
	episodes, err := c.inner.GetEpisodes(ctx, tvdbID)
	if err != nil {
		return nil, err
	}
	c.set(key, episodes, c.opts.EpisodesTTL)
	return episodes, nil
}

// get retrieves a non-expired value from the cache.
func (c *Source) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.cache[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// set stores value in the cache with the given TTL.
func (c *Source) set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// sweepLoop runs on a ticker and purges expired entries until Stop is called.
func (c *Source) sweepLoop() {
	ticker := time.NewTicker(c.opts.SweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.sweep()
		case <-c.stopCh:
			return
		}
	}
}

// sweep removes all expired entries from the cache under a write lock.
func (c *Source) sweep() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.cache {
		if now.After(e.expiresAt) {
			delete(c.cache, k)
		}
	}
}
