// Package notification defines the Notification interface and supporting types.
// A Notification is a Provider that can dispatch event messages to an external
// service (e.g. Discord, Slack, Telegram).
package notification

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/providers"
)

// GrabMessage carries details about a release that was grabbed by the downloader.
type GrabMessage struct {
	SeriesTitle  string
	EpisodeTitle string
	Quality      string
	Indexer      string
}

// DownloadMessage carries details about a completed download.
type DownloadMessage struct {
	SeriesTitle  string
	EpisodeTitle string
	Quality      string
}

// HealthMessage carries details about a health-check event.
type HealthMessage struct {
	Type    string
	Message string
}

// Notification extends Provider with notification-specific event methods.
type Notification interface {
	providers.Provider
	OnGrab(ctx context.Context, msg GrabMessage) error
	OnDownload(ctx context.Context, msg DownloadMessage) error
	OnHealthIssue(ctx context.Context, msg HealthMessage) error
}
