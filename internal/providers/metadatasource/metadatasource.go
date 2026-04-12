// Package metadatasource defines the abstract interface and shared types for
// fetching series and episode metadata from an external provider.
package metadatasource

import (
	"context"
	"time"
)

// SeriesSearchResult is a brief result from searching for a series by title.
type SeriesSearchResult struct {
	TvdbID   int64
	Title    string
	Year     int
	Overview string
	Status   string
	Network  string
	Slug     string
}

// SeriesInfo is the full metadata for a single series.
type SeriesInfo struct {
	TvdbID   int64
	Title    string
	Year     int
	Overview string
	Status   string // Continuing, Ended, Upcoming
	Network  string
	Runtime  int
	AirTime  string
	Slug     string
	Genres   []string
}

// EpisodeInfo is metadata for a single episode.
type EpisodeInfo struct {
	TvdbID                int64
	SeasonNumber          int
	EpisodeNumber         int
	AbsoluteEpisodeNumber *int
	Title                 string
	Overview              string
	AirDate               *time.Time
}

// MetadataSource fetches series and episode information from an external
// provider (TVDB, TMDb, TVMaze, etc.).
type MetadataSource interface {
	// SearchSeries finds series matching the query string.
	SearchSeries(ctx context.Context, query string) ([]SeriesSearchResult, error)

	// GetSeries returns full metadata for a series by TVDB ID.
	GetSeries(ctx context.Context, tvdbID int64) (SeriesInfo, error)

	// GetEpisodes returns all episodes for a series.
	GetEpisodes(ctx context.Context, tvdbID int64) ([]EpisodeInfo, error)
}
