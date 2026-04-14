// Package notifiarr implements a notification.Notification that posts to
// Notifiarr's API endpoint at https://notifiarr.com/api/v1/notification/sonarr.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Notifiarr/).
package notifiarr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

const apiURL = "https://notifiarr.com/api/v1/notification/sonarr"

// Notifiarr sends events to a user's Notifiarr account for central routing.
type Notifiarr struct {
	settings Settings
	client   *http.Client
}

// New constructs a Notifiarr provider.
func New(settings Settings, client *http.Client) *Notifiarr {
	return &Notifiarr{settings: settings, client: client}
}

func (n *Notifiarr) Implementation() string { return "Notifiarr" }
func (n *Notifiarr) DefaultName() string    { return "Notifiarr" }
func (n *Notifiarr) Settings() any          { return &n.settings }

func (n *Notifiarr) Test(ctx context.Context) error {
	return n.send(ctx, "Test", map[string]any{
		"event":       "Test",
		"instanceName": "sonarr2",
	})
}

func (n *Notifiarr) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return n.send(ctx, "Grab", map[string]any{
		"event":        "Grab",
		"instanceName": "sonarr2",
		"series":       msg.SeriesTitle,
		"episode":      msg.EpisodeTitle,
		"quality":      msg.Quality,
		"indexer":      msg.Indexer,
	})
}

func (n *Notifiarr) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return n.send(ctx, "Download", map[string]any{
		"event":        "Download",
		"instanceName": "sonarr2",
		"series":       msg.SeriesTitle,
		"episode":      msg.EpisodeTitle,
		"quality":      msg.Quality,
	})
}

func (n *Notifiarr) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return n.send(ctx, "Health", map[string]any{
		"event":        "Health",
		"instanceName": "sonarr2",
		"type":         msg.Type,
		"message":      msg.Message,
	})
}

func (n *Notifiarr) send(ctx context.Context, _ string, payload map[string]any) error {
	if n.settings.APIKey == "" {
		return fmt.Errorf("notifiarr: APIKey is not configured")
	}
	client := n.client
	if client == nil {
		client = http.DefaultClient
	}
	return notification.PostJSONWithHeaders(ctx, client, apiURL, payload, map[string]string{
		"X-API-Key": n.settings.APIKey,
	})
}
