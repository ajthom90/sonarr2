package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// stubSource is a test double for metadatasource.MetadataSource.
type stubSource struct {
	series   metadatasource.SeriesInfo
	episodes []metadatasource.EpisodeInfo
}

func (s *stubSource) SearchSeries(ctx context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	return nil, nil
}

func (s *stubSource) GetSeries(ctx context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	return s.series, nil
}

func (s *stubSource) GetEpisodes(ctx context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	return s.episodes, nil
}

// setupHandlerLib opens an in-memory SQLite library with migrations applied.
func setupHandlerLib(t *testing.T) (*library.Library, *db.SQLitePool) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	lib, err := library.New(pool, events.NewNoopBus())
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return lib, pool
}

// makeCmd builds a Command with JSON body {"seriesId": id}.
func makeCmd(t *testing.T, seriesID int64) commands.Command {
	t.Helper()
	body, err := json.Marshal(map[string]int64{"seriesId": seriesID})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return commands.Command{Name: "RefreshSeriesMetadata", Body: body}
}

func TestRefreshSeriesHandlerCreatesEpisodes(t *testing.T) {
	ctx := context.Background()
	lib, _ := setupHandlerLib(t)

	// Create a series in the library.
	s, err := lib.Series.Create(ctx, library.Series{
		TvdbID:     42,
		Title:      "Original Title",
		Slug:       "original-title",
		Status:     "continuing",
		SeriesType: "standard",
		Path:       "/tv/Original Title",
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	airDate := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)
	stub := &stubSource{
		series: metadatasource.SeriesInfo{
			TvdbID:  42,
			Title:   "Updated Title",
			Status:  "ended",
			Network: "HBO",
		},
		episodes: []metadatasource.EpisodeInfo{
			{TvdbID: 101, SeasonNumber: 1, EpisodeNumber: 1, Title: "Pilot", AirDate: &airDate},
			{TvdbID: 102, SeasonNumber: 1, EpisodeNumber: 2, Title: "Episode 2"},
			{TvdbID: 103, SeasonNumber: 2, EpisodeNumber: 1, Title: "Season 2 Premiere"},
		},
	}

	handler := NewRefreshSeriesHandler(stub, lib)
	if err := handler.Handle(ctx, makeCmd(t, s.ID)); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify series metadata was updated.
	updated, err := lib.Series.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get series: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want Updated Title", updated.Title)
	}
	if updated.Status != "ended" {
		t.Errorf("Status = %q, want ended", updated.Status)
	}

	// Verify 3 episodes were created.
	eps, err := lib.Episodes.ListForSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(eps) != 3 {
		t.Errorf("episode count = %d, want 3", len(eps))
	}

	// Verify seasons were created (season 1 and season 2).
	seasons, err := lib.Seasons.ListForSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListForSeries seasons: %v", err)
	}
	if len(seasons) != 2 {
		t.Errorf("season count = %d, want 2", len(seasons))
	}
	for _, season := range seasons {
		if !season.Monitored {
			t.Errorf("season %d Monitored = false, want true", season.SeasonNumber)
		}
	}
}

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

func TestRefreshSeriesHandlerUpdatesExistingEpisodes(t *testing.T) {
	ctx := context.Background()
	lib, _ := setupHandlerLib(t)

	// Create a series.
	s, err := lib.Series.Create(ctx, library.Series{
		TvdbID:     99,
		Title:      "My Show",
		Slug:       "my-show",
		Status:     "continuing",
		SeriesType: "standard",
		Path:       "/tv/My Show",
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	// Pre-create 2 episodes with original titles.
	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      s.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Old Title 1",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode 1: %v", err)
	}
	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      s.ID,
		SeasonNumber:  1,
		EpisodeNumber: 2,
		Title:         "Old Title 2",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode 2: %v", err)
	}

	// Configure stub: return updated titles for the 2 existing episodes
	// plus 1 new episode.
	stub := &stubSource{
		series: metadatasource.SeriesInfo{
			TvdbID: 99,
			Title:  "My Show",
			Status: "continuing",
		},
		episodes: []metadatasource.EpisodeInfo{
			{TvdbID: 201, SeasonNumber: 1, EpisodeNumber: 1, Title: "New Title 1"},
			{TvdbID: 202, SeasonNumber: 1, EpisodeNumber: 2, Title: "New Title 2"},
			{TvdbID: 203, SeasonNumber: 1, EpisodeNumber: 3, Title: "Brand New Episode"},
		},
	}

	handler := NewRefreshSeriesHandler(stub, lib)
	if err := handler.Handle(ctx, makeCmd(t, s.ID)); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	eps, err := lib.Episodes.ListForSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}

	// Expect exactly 3 episodes total (2 updated + 1 new).
	if len(eps) != 3 {
		t.Errorf("episode count = %d, want 3", len(eps))
	}

	// Build a title map for easy assertion.
	titleByKey := map[string]string{}
	for _, ep := range eps {
		key := fmt.Sprintf("%d-%d", ep.SeasonNumber, ep.EpisodeNumber)
		titleByKey[key] = ep.Title
	}

	cases := []struct {
		key   string
		title string
	}{
		{"1-1", "New Title 1"},
		{"1-2", "New Title 2"},
		{"1-3", "Brand New Episode"},
	}
	for _, c := range cases {
		if got, ok := titleByKey[c.key]; !ok {
			t.Errorf("episode %s not found", c.key)
		} else if got != c.title {
			t.Errorf("episode %s title = %q, want %q", c.key, got, c.title)
		}
	}
}
