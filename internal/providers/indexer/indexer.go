// Package indexer defines the Indexer interface and supporting types.
// An Indexer is a Provider that can search for releases and poll an RSS feed.
package indexer

import (
	"context"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers"
)

// DownloadProtocol identifies whether a release is delivered over usenet or
// via a torrent tracker.
type DownloadProtocol string

const (
	ProtocolUsenet  DownloadProtocol = "usenet"
	ProtocolTorrent DownloadProtocol = "torrent"
)

// Release is a single result from an indexer search or RSS feed.
type Release struct {
	Title       string
	GUID        string
	DownloadURL string
	InfoURL     string
	Size        int64
	PublishDate time.Time
	Indexer     string
	Protocol    DownloadProtocol
	Seeders     int
	Leechers    int
	Categories  []int
}

// SearchRequest describes what to search for.
type SearchRequest struct {
	SeriesTitle string
	TvdbID      int64
	Season      int
	Episode     int
	Categories  []int
}

// Indexer extends Provider with indexer-specific methods.
type Indexer interface {
	providers.Provider
	Protocol() DownloadProtocol
	SupportsRss() bool
	SupportsSearch() bool
	FetchRss(ctx context.Context) ([]Release, error)
	Search(ctx context.Context, req SearchRequest) ([]Release, error)
}
