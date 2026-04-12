package library

import (
	"context"
	"time"
)

// EpisodeFile represents a file on disk holding one or more episodes.
// For M2 we keep the model minimal — quality is a plain string, media
// info is absent, custom formats are not tracked. Later milestones
// expand the struct as they need more fields.
type EpisodeFile struct {
	ID           int64
	SeriesID     int64
	SeasonNumber int32
	RelativePath string
	Size         int64
	DateAdded    time.Time
	ReleaseGroup string
	QualityName  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// EpisodeFilesStore reads and writes EpisodeFile rows.
type EpisodeFilesStore interface {
	Create(ctx context.Context, f EpisodeFile) (EpisodeFile, error)
	Get(ctx context.Context, id int64) (EpisodeFile, error)
	ListForSeries(ctx context.Context, seriesID int64) ([]EpisodeFile, error)
	Delete(ctx context.Context, id int64) error
	// SumSizesForSeries returns (file_count, total_size_bytes) for a series.
	// Used by SeriesStatsStore.Recompute.
	SumSizesForSeries(ctx context.Context, seriesID int64) (count int, sizeOnDisk int64, err error)
}

// EpisodeFileAdded is published by EpisodeFilesStore.Create.
type EpisodeFileAdded struct {
	ID       int64
	SeriesID int64
	Size     int64
}

// EpisodeFileDeleted is published by EpisodeFilesStore.Delete.
type EpisodeFileDeleted struct {
	ID       int64
	SeriesID int64
}
