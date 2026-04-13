// Package history records grab, import, and failure events for episodes.
// It provides a Store interface with Postgres and SQLite implementations
// backed by sqlc-generated query code.
package history

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// EventType identifies the kind of history event recorded.
type EventType string

const (
	EventGrabbed          EventType = "grabbed"
	EventDownloadImported EventType = "downloadImported"
	EventDownloadFailed   EventType = "downloadFailed"
	EventEpisodeRenamed   EventType = "episodeRenamed"
	EventEpisodeDeleted   EventType = "episodeFileDeleted"
)

// Entry is a single grab/import/failure event for an episode.
type Entry struct {
	ID          int64
	EpisodeID   int64
	SeriesID    int64
	SourceTitle string
	QualityName string
	EventType   EventType
	Date        time.Time
	DownloadID  string
	Data        json.RawMessage
}

// Store provides read/write access to history entries.
type Store interface {
	// Create persists a new history entry and returns it with the assigned ID
	// and server-generated date.
	Create(ctx context.Context, entry Entry) (Entry, error)

	// ListForSeries returns all history entries for the given series, ordered
	// by date descending.
	ListForSeries(ctx context.Context, seriesID int64) ([]Entry, error)

	// ListForEpisode returns all history entries for the given episode,
	// ordered by date descending.
	ListForEpisode(ctx context.Context, episodeID int64) ([]Entry, error)

	// FindByDownloadID returns history entries whose download_id matches,
	// ordered by date descending.
	FindByDownloadID(ctx context.Context, downloadID string) ([]Entry, error)

	// DeleteForSeries deletes all history entries for the given series.
	DeleteForSeries(ctx context.Context, seriesID int64) error

	// ListAll returns all history entries ordered by date descending, used for
	// paged API responses.
	ListAll(ctx context.Context) ([]Entry, error)

	// DeleteBefore removes all history entries with a date before the given time.
	// Returns the number of deleted entries.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// ErrNotFound is returned when a requested history entry does not exist.
var ErrNotFound = errors.New("history: not found")
