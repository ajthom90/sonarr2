// Package library owns the TV catalog domain — series, seasons, episodes,
// episode files, and cached series statistics. All entities live in one
// package because they share foreign-key relationships and statistics
// depend on joins across episodes and episode_files. Each entity provides
// a canonical Go struct, a Store interface, and Postgres + SQLite
// implementations wrapping sqlc-generated code. Store constructors take
// an events.Bus so Create/Update/Delete can publish typed domain events.
package library

import (
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
)

// ErrNotFound is returned by Store methods when a lookup finds nothing.
var ErrNotFound = errors.New("library: not found")

// Library bundles all catalog Store implementations for a given dialect.
// Obtained via library.New(pool, bus). Tasks 4-7 add Seasons, Episodes,
// EpisodeFiles, and Stats; during the interim (Task 3 through Task 7),
// some fields will be nil — consumers must not access them until the
// corresponding task lands.
type Library struct {
	Series       SeriesStore
	Seasons      SeasonsStore
	Episodes     EpisodesStore
	EpisodeFiles EpisodeFilesStore
	Stats        SeriesStatsStore
}

// New constructs a Library backed by the given pool. Dispatches on the
// pool's concrete type to pick the right dialect-specific Store
// implementations. Returns an error if the pool type is not recognized.
func New(pool db.Pool, bus events.Bus) (*Library, error) {
	switch p := pool.(type) {
	case *db.PostgresPool:
		return &Library{
			Series:  newPostgresSeriesStore(p, bus),
			Seasons: newPostgresSeasonsStore(p, bus),
		}, nil
	case *db.SQLitePool:
		return &Library{
			Series:  newSqliteSeriesStore(p, bus),
			Seasons: newSqliteSeasonsStore(p, bus),
		}, nil
	default:
		return nil, fmt.Errorf("library: unsupported pool type %T", pool)
	}
}
