// Package blocklist provides storage for release blocklists.
//
// The blocklist records releases that should not be grabbed again — either
// because the user manually blocked them, or because an import failed and
// auto-blocklisting is enabled. The decision engine consults the blocklist
// when evaluating releases and rejects any that match a blocklisted source
// title for the same series.
//
// Ported from Sonarr (src/NzbDrone.Core/Blocklisting/).
package blocklist

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a blocklist entry does not exist.
var ErrNotFound = errors.New("blocklist: not found")

// Protocol enumerates the release delivery protocols that Sonarr distinguishes.
type Protocol string

const (
	ProtocolUnknown Protocol = ""
	ProtocolUsenet  Protocol = "usenet"
	ProtocolTorrent Protocol = "torrent"
)

// Entry is one row of the blocklist. Field names mirror Sonarr's Blocklist
// model (src/NzbDrone.Core/Blocklisting/Blocklist.cs).
type Entry struct {
	ID              int
	SeriesID        int
	EpisodeIDs      []int
	SourceTitle     string
	Quality         []byte // opaque QualityModel JSON
	Languages       []byte // opaque Language[] JSON
	Date            time.Time
	PublishedDate   *time.Time
	Size            *int64
	Protocol        Protocol
	Indexer         string
	IndexerFlags    int
	ReleaseType     string
	Message         string
	TorrentInfoHash string
}

// Page is the paged result shape used by the v3 API.
type Page struct {
	Page         int
	PageSize     int
	TotalRecords int
	Records      []Entry
}

// Store provides CRUD + paged query access to blocklist entries.
type Store interface {
	Create(ctx context.Context, e Entry) (Entry, error)
	GetByID(ctx context.Context, id int) (Entry, error)
	List(ctx context.Context, page, pageSize int) (Page, error)
	ListBySeries(ctx context.Context, seriesID int) ([]Entry, error)
	Delete(ctx context.Context, id int) error
	DeleteMany(ctx context.Context, ids []int) error
	DeleteBySeries(ctx context.Context, seriesID int) error
	Clear(ctx context.Context) error
}

// Matches reports whether the given release title is blocklisted for the
// specified series. Callers typically use this from the decision engine.
func Matches(entries []Entry, seriesID int, releaseTitle string) bool {
	for _, e := range entries {
		if e.SeriesID == seriesID && e.SourceTitle == releaseTitle {
			return true
		}
	}
	return false
}
