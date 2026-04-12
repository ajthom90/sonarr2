package library

import (
	"context"
	"time"
)

// SeriesStatistics is a cached roll-up of episode/file counts and disk
// usage for a single series. Recomputed reactively by the app's event
// subscribers when episodes or episode files change.
type SeriesStatistics struct {
	SeriesID              int64
	EpisodeCount          int32
	EpisodeFileCount      int32
	MonitoredEpisodeCount int32
	SizeOnDisk            int64
	UpdatedAt             time.Time
}

// SeriesStatsStore reads and writes series_statistics rows. Recompute
// derives counts from episodes + episode_files and upserts the row.
type SeriesStatsStore interface {
	Get(ctx context.Context, seriesID int64) (SeriesStatistics, error)
	Recompute(ctx context.Context, seriesID int64) error
	Delete(ctx context.Context, seriesID int64) error
}

// recomputeHelper is the dialect-agnostic part of Recompute. Both the
// Postgres and SQLite implementations load the counts via the entity
// Stores passed in, then call the dialect's Upsert.
type recomputeHelper struct {
	episodes EpisodesStore
	files    EpisodeFilesStore
}

// derive computes the counts for a series by querying the entity stores.
// Called by each dialect's Recompute.
func (h *recomputeHelper) derive(ctx context.Context, seriesID int64) (SeriesStatistics, error) {
	total, monitored, err := h.episodes.CountForSeries(ctx, seriesID)
	if err != nil {
		return SeriesStatistics{}, err
	}
	fileCount, sizeOnDisk, err := h.files.SumSizesForSeries(ctx, seriesID)
	if err != nil {
		return SeriesStatistics{}, err
	}
	return SeriesStatistics{
		SeriesID:              seriesID,
		EpisodeCount:          int32(total),
		MonitoredEpisodeCount: int32(monitored),
		EpisodeFileCount:      int32(fileCount),
		SizeOnDisk:            sizeOnDisk,
	}, nil
}
