package cached_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/cached"
)

// mockSource is a MetadataSource that counts calls to each method.
type mockSource struct {
	searchCalls  atomic.Int32
	getCalls     atomic.Int32
	episodeCalls atomic.Int32
}

func (m *mockSource) SearchSeries(_ context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	m.searchCalls.Add(1)
	return []metadatasource.SeriesSearchResult{
		{TvdbID: 1, Title: "Result for " + query},
	}, nil
}

func (m *mockSource) GetSeries(_ context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	m.getCalls.Add(1)
	return metadatasource.SeriesInfo{TvdbID: tvdbID, Title: "Series"}, nil
}

func (m *mockSource) GetEpisodes(_ context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	m.episodeCalls.Add(1)
	return []metadatasource.EpisodeInfo{
		{TvdbID: tvdbID, SeasonNumber: 1, EpisodeNumber: 1, Title: "Pilot"},
	}, nil
}

// TestCacheMissThenHit verifies that a second call with the same ID
// is served from cache and the inner source is called only once.
func TestCacheMissThenHit(t *testing.T) {
	mock := &mockSource{}
	c := cached.New(mock, cached.Options{})
	defer c.Stop()

	ctx := context.Background()
	const id int64 = 71663

	// First call — cache miss, must hit inner.
	info1, err := c.GetSeries(ctx, id)
	if err != nil {
		t.Fatalf("first GetSeries: %v", err)
	}
	if info1.TvdbID != id {
		t.Errorf("TvdbID = %d, want %d", info1.TvdbID, id)
	}

	// Second call — cache hit, inner must not be called again.
	info2, err := c.GetSeries(ctx, id)
	if err != nil {
		t.Fatalf("second GetSeries: %v", err)
	}
	if info2.TvdbID != id {
		t.Errorf("TvdbID = %d, want %d", info2.TvdbID, id)
	}

	if got := mock.getCalls.Load(); got != 1 {
		t.Errorf("inner called %d times, want 1", got)
	}
}

// TestCacheExpiry verifies that a cache entry is not served after its TTL expires.
func TestCacheExpiry(t *testing.T) {
	mock := &mockSource{}
	c := cached.New(mock, cached.Options{
		SeriesTTL:   50 * time.Millisecond,
		EpisodesTTL: 50 * time.Millisecond,
		SearchTTL:   50 * time.Millisecond,
		SweepEvery:  10 * time.Millisecond,
	})
	defer c.Stop()

	ctx := context.Background()
	const id int64 = 42

	// Populate the cache.
	if _, err := c.GetSeries(ctx, id); err != nil {
		t.Fatalf("first GetSeries: %v", err)
	}
	if n := mock.getCalls.Load(); n != 1 {
		t.Fatalf("expected 1 call after first Get, got %d", n)
	}

	// Wait for TTL to expire.
	time.Sleep(80 * time.Millisecond)

	// Second call should miss the expired cache.
	if _, err := c.GetSeries(ctx, id); err != nil {
		t.Fatalf("second GetSeries: %v", err)
	}
	if n := mock.getCalls.Load(); n != 2 {
		t.Errorf("expected 2 inner calls after expiry, got %d", n)
	}
}

// TestCacheInvalidate verifies that Invalidate removes both series and episodes
// entries so that subsequent calls re-fetch from the inner source.
func TestCacheInvalidate(t *testing.T) {
	mock := &mockSource{}
	c := cached.New(mock, cached.Options{})
	defer c.Stop()

	ctx := context.Background()
	const id int64 = 99

	// Populate both caches.
	if _, err := c.GetSeries(ctx, id); err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	if _, err := c.GetEpisodes(ctx, id); err != nil {
		t.Fatalf("GetEpisodes: %v", err)
	}
	if n := mock.getCalls.Load(); n != 1 {
		t.Fatalf("expected 1 GetSeries call, got %d", n)
	}
	if n := mock.episodeCalls.Load(); n != 1 {
		t.Fatalf("expected 1 GetEpisodes call, got %d", n)
	}

	// Invalidate the series entry.
	c.Invalidate(id)

	// Both should re-fetch after invalidation.
	if _, err := c.GetSeries(ctx, id); err != nil {
		t.Fatalf("GetSeries post-invalidate: %v", err)
	}
	if _, err := c.GetEpisodes(ctx, id); err != nil {
		t.Fatalf("GetEpisodes post-invalidate: %v", err)
	}
	if n := mock.getCalls.Load(); n != 2 {
		t.Errorf("GetSeries inner calls = %d, want 2", n)
	}
	if n := mock.episodeCalls.Load(); n != 2 {
		t.Errorf("GetEpisodes inner calls = %d, want 2", n)
	}
}

// TestCacheSearchSeries verifies cache miss, hit, and that a different query
// triggers a new miss.
func TestCacheSearchSeries(t *testing.T) {
	mock := &mockSource{}
	c := cached.New(mock, cached.Options{})
	defer c.Stop()

	ctx := context.Background()

	// First search — miss.
	if _, err := c.SearchSeries(ctx, "breaking bad"); err != nil {
		t.Fatalf("first search: %v", err)
	}
	if n := mock.searchCalls.Load(); n != 1 {
		t.Fatalf("expected 1 search call, got %d", n)
	}

	// Same query — hit.
	if _, err := c.SearchSeries(ctx, "breaking bad"); err != nil {
		t.Fatalf("second search: %v", err)
	}
	if n := mock.searchCalls.Load(); n != 1 {
		t.Errorf("expected still 1 search call after hit, got %d", n)
	}

	// Different query — new miss.
	if _, err := c.SearchSeries(ctx, "the wire"); err != nil {
		t.Fatalf("third search: %v", err)
	}
	if n := mock.searchCalls.Load(); n != 2 {
		t.Errorf("expected 2 search calls after new query, got %d", n)
	}
}

// TestCacheConcurrent exercises Get/Search/Invalidate concurrently to catch
// data races when run with -race.
func TestCacheConcurrent(t *testing.T) {
	mock := &mockSource{}
	c := cached.New(mock, cached.Options{
		SweepEvery: 5 * time.Millisecond,
	})
	defer c.Stop()

	ctx := context.Background()
	const workers = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func(n int) {
			defer wg.Done()
			id := int64(n%5) + 1 // 5 distinct IDs so there's contention

			if _, err := c.GetSeries(ctx, id); err != nil {
				t.Errorf("GetSeries(%d): %v", id, err)
			}
			if _, err := c.GetEpisodes(ctx, id); err != nil {
				t.Errorf("GetEpisodes(%d): %v", id, err)
			}
			if _, err := c.SearchSeries(ctx, "query"); err != nil {
				t.Errorf("SearchSeries: %v", err)
			}
			// Occasionally invalidate to exercise write paths.
			if n%10 == 0 {
				c.Invalidate(id)
			}
		}(i)
	}

	wg.Wait()
}
