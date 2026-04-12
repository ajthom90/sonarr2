// Package library owns the TV catalog domain — series, seasons, episodes,
// episode files, and cached series statistics. All entities live in one
// package because they share foreign-key relationships and statistics
// depend on joins across episodes and episode_files. Each entity provides
// a canonical Go struct, a Store interface, and Postgres + SQLite
// implementations wrapping sqlc-generated code. Store constructors take
// an events.Bus so Create/Update/Delete can publish typed domain events.
package library

import "errors"

// ErrNotFound is returned by Store methods when a lookup finds nothing.
var ErrNotFound = errors.New("library: not found")

// Library bundles all catalog Store implementations for a given dialect.
// Obtained via library.New(pool, bus).
type Library struct {
	Series       SeriesStore
	Seasons      SeasonsStore
	Episodes     EpisodesStore
	EpisodeFiles EpisodeFilesStore
	Stats        SeriesStatsStore
}
