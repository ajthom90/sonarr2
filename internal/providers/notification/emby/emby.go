// Package emby implements a notification.Notification for Emby / MediaBrowser.
// Sends notifications via POST /Notifications/Admin and triggers library
// refresh via POST /Library/Refresh.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Emby/).
package emby

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Emby represents a single Emby/MediaBrowser instance.
type Emby struct {
	settings Settings
	client   *http.Client
}

// New constructs an Emby provider.
func New(settings Settings, client *http.Client) *Emby {
	return &Emby{settings: settings, client: client}
}

func (e *Emby) Implementation() string { return "MediaBrowser" }
func (e *Emby) DefaultName() string    { return "Emby" }
func (e *Emby) Settings() any          { return &e.settings }

func (e *Emby) Test(ctx context.Context) error {
	return notification.Get(ctx, e.client, e.url("/System/Info"), e.headers())
}

func (e *Emby) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	if !e.settings.Notify {
		return nil
	}
	return e.sendMessage(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (e *Emby) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	if e.settings.Notify {
		if err := e.sendMessage(ctx, "Sonarr - Download Complete",
			fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle)); err != nil {
			return err
		}
	}
	if e.settings.UpdateLibrary {
		return notification.PostJSON(ctx, e.client, e.url("/Library/Refresh"), nil)
	}
	return nil
}

func (e *Emby) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	if !e.settings.Notify {
		return nil
	}
	return e.sendMessage(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (e *Emby) sendMessage(ctx context.Context, name, desc string) error {
	return notification.PostJSON(ctx, e.client, e.url("/Notifications/Admin"),
		map[string]any{"Name": name, "Description": desc})
}

func (e *Emby) url(path string) string {
	scheme := "http"
	if e.settings.UseSSL {
		scheme = "https"
	}
	port := e.settings.Port
	if port == 0 {
		port = 8096
	}
	return fmt.Sprintf("%s://%s:%d/emby%s?api_key=%s",
		scheme, e.settings.Host, port, path, e.settings.APIKey)
}

func (e *Emby) headers() map[string]string {
	return map[string]string{
		"X-Emby-Token": e.settings.APIKey,
		"Accept":       "application/json",
	}
}
