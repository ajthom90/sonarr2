package metadatasource_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// stubSource is a minimal implementation of MetadataSource used to verify
// the interface is satisfiable.
type stubSource struct{}

func (s *stubSource) SearchSeries(_ context.Context, _ string) ([]metadatasource.SeriesSearchResult, error) {
	abs := 1
	_ = metadatasource.EpisodeInfo{AbsoluteEpisodeNumber: &abs}
	return []metadatasource.SeriesSearchResult{
		{TvdbID: 1, Title: "Test", Year: 2024, Overview: "overview", Status: "Continuing", Network: "ABC", Slug: "test"},
	}, nil
}

func (s *stubSource) GetSeries(_ context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	return metadatasource.SeriesInfo{
		TvdbID:   tvdbID,
		Title:    "Test Series",
		Year:     2024,
		Overview: "A test series",
		Status:   "Continuing",
		Network:  "ABC",
		Runtime:  30,
		AirTime:  "20:00",
		Slug:     "test-series",
		Genres:   []string{"Drama"},
	}, nil
}

func (s *stubSource) GetEpisodes(_ context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	aired := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	abs := 1
	return []metadatasource.EpisodeInfo{
		{
			TvdbID:                100,
			SeasonNumber:          1,
			EpisodeNumber:         1,
			AbsoluteEpisodeNumber: &abs,
			Title:                 "Pilot",
			Overview:              "The first episode",
			AirDate:               &aired,
		},
		{
			TvdbID:        101,
			SeasonNumber:  1,
			EpisodeNumber: 2,
			Title:         "Episode 2",
			AirDate:       nil,
		},
	}, nil
}

// TestStubSatisfiesInterface verifies that a concrete type can implement
// MetadataSource and that all three method signatures compile correctly.
func TestStubSatisfiesInterface(t *testing.T) {
	var src metadatasource.MetadataSource = &stubSource{}

	ctx := context.Background()

	results, err := src.SearchSeries(ctx, "test")
	if err != nil {
		t.Fatalf("SearchSeries: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SearchSeries: expected at least one result")
	}
	if results[0].TvdbID == 0 {
		t.Error("SearchSeries: expected non-zero TvdbID")
	}

	info, err := src.GetSeries(ctx, 71663)
	if err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	if info.TvdbID != 71663 {
		t.Errorf("GetSeries: TvdbID = %d, want 71663", info.TvdbID)
	}
	if len(info.Genres) == 0 {
		t.Error("GetSeries: expected genres")
	}

	episodes, err := src.GetEpisodes(ctx, 71663)
	if err != nil {
		t.Fatalf("GetEpisodes: %v", err)
	}
	if len(episodes) < 2 {
		t.Fatalf("GetEpisodes: expected at least 2 episodes, got %d", len(episodes))
	}
	if episodes[0].AbsoluteEpisodeNumber == nil {
		t.Error("GetEpisodes[0]: expected non-nil AbsoluteEpisodeNumber")
	}
	if episodes[1].AirDate != nil {
		t.Error("GetEpisodes[1]: expected nil AirDate")
	}
}
