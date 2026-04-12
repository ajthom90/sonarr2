// Package downloadclient defines the DownloadClient interface and supporting
// types. A DownloadClient is a Provider that can accept releases for download,
// report queue status, and remove items.
package downloadclient

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/providers"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Item represents an active or recently completed download in the client.
type Item struct {
	DownloadID string
	Title      string
	Status     string // queued, downloading, completed, failed, paused
	TotalSize  int64
	Remaining  int64
	OutputPath string
}

// Status reports the download client's overall state.
type Status struct {
	IsLocalhost       bool
	OutputRootFolders []string
}

// DownloadClient extends Provider with download-client-specific methods.
type DownloadClient interface {
	providers.Provider
	Protocol() indexer.DownloadProtocol
	Add(ctx context.Context, url string, title string) (downloadID string, err error)
	Items(ctx context.Context) ([]Item, error)
	Remove(ctx context.Context, downloadID string, deleteData bool) error
	Status(ctx context.Context) (Status, error)
}
