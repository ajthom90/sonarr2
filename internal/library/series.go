package library

import (
	"context"
	"time"
)

// Series is a TV show tracked by sonarr2. It corresponds to one row in
// the `series` table.
type Series struct {
	ID               int64
	TvdbID           int64
	Title            string
	Slug             string
	Status           string // continuing | ended | upcoming
	SeriesType       string // standard | daily | anime
	Path             string
	Monitored        bool
	QualityProfileID int64 // 0 = unassigned
	SeasonFolder     bool
	MonitorNewItems  string // "all" | "none"
	Added            time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SeriesStore reads and writes Series rows. Create, Update, and Delete
// publish typed events to the events.Bus passed to the constructor.
type SeriesStore interface {
	Create(ctx context.Context, s Series) (Series, error)
	Get(ctx context.Context, id int64) (Series, error)
	GetByTvdbID(ctx context.Context, tvdbID int64) (Series, error)
	GetBySlug(ctx context.Context, slug string) (Series, error)
	List(ctx context.Context) ([]Series, error)
	Update(ctx context.Context, s Series) error
	Delete(ctx context.Context, id int64) error
}

// SeriesAdded is published by SeriesStore.Create.
type SeriesAdded struct {
	ID     int64
	TvdbID int64
	Title  string
}

// SeriesUpdated is published by SeriesStore.Update.
type SeriesUpdated struct {
	ID     int64
	TvdbID int64
}

// SeriesDeleted is published by SeriesStore.Delete.
type SeriesDeleted struct {
	ID int64
}
