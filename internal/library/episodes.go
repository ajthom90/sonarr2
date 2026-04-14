package library

import (
	"context"
	"time"
)

// Episode is one episode of a Series. It carries a nullable absolute
// episode number (used by anime), a nullable air date, and a nullable
// reference to an episode_files row via EpisodeFileID.
type Episode struct {
	ID                    int64
	SeriesID              int64
	SeasonNumber          int32
	EpisodeNumber         int32
	AbsoluteEpisodeNumber *int32 // nil when not applicable
	Title                 string
	Overview              string
	AirDateUtc            *time.Time // nil when unknown
	Monitored             bool
	EpisodeFileID         *int64 // nil when no file has been imported
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// EpisodesStore reads and writes Episode rows.
type EpisodesStore interface {
	Create(ctx context.Context, e Episode) (Episode, error)
	Get(ctx context.Context, id int64) (Episode, error)
	ListForSeries(ctx context.Context, seriesID int64) ([]Episode, error)
	// ListAll returns every episode across all series, ordered by air_date_utc.
	// Used by the calendar and wanted/missing API endpoints.
	ListAll(ctx context.Context) ([]Episode, error)
	Update(ctx context.Context, e Episode) error
	// SetMonitored toggles just the monitored flag for a single episode.
	// Used by library.ApplyMonitorMode to avoid a full Update round-trip
	// per episode when applying a series-wide monitor rule.
	SetMonitored(ctx context.Context, episodeID int64, monitored bool) error
	Delete(ctx context.Context, id int64) error
	// CountForSeries returns (total, monitored) episode counts for a series.
	// Used by SeriesStatsStore.Recompute.
	CountForSeries(ctx context.Context, seriesID int64) (total int, monitored int, err error)
}

// EpisodeAdded is published by EpisodesStore.Create.
type EpisodeAdded struct {
	ID       int64
	SeriesID int64
}

// EpisodeUpdated is published by EpisodesStore.Update.
type EpisodeUpdated struct {
	ID       int64
	SeriesID int64
}

// EpisodeDeleted is published by EpisodesStore.Delete.
type EpisodeDeleted struct {
	ID       int64
	SeriesID int64
}
