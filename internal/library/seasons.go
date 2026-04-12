package library

import "context"

// Season represents a single season of a Series. Seasons have a composite
// primary key (series_id, season_number) because they're a thin join table.
type Season struct {
	SeriesID     int64
	SeasonNumber int32
	Monitored    bool
}

// SeasonsStore reads and writes Season rows. Upsert is the canonical write
// operation — seasons don't have an independent identity so we don't have
// Create/Update separately.
type SeasonsStore interface {
	Upsert(ctx context.Context, s Season) error
	Get(ctx context.Context, seriesID int64, seasonNumber int32) (Season, error)
	ListForSeries(ctx context.Context, seriesID int64) ([]Season, error)
	Delete(ctx context.Context, seriesID int64, seasonNumber int32) error
}

// SeasonUpdated is published by SeasonsStore.Upsert.
type SeasonUpdated struct {
	SeriesID     int64
	SeasonNumber int32
	Monitored    bool
}
