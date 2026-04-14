// Package plex implements a notification.Notification that triggers a Plex
// library update over the Plex Media Server HTTP API when a new episode is
// imported.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Plex/).
package plex

import (
	"context"
	"fmt"
	"net/url"

	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Plex updates the Plex library when notified.
type Plex struct {
	settings Settings
	client   *http.Client
}

// New constructs a Plex provider.
func New(settings Settings, client *http.Client) *Plex {
	return &Plex{settings: settings, client: client}
}

func (p *Plex) Implementation() string { return "PlexServer" }
func (p *Plex) DefaultName() string    { return "Plex Media Server" }
func (p *Plex) Settings() any          { return &p.settings }

// Test verifies Plex is reachable with the given token (GET /identity).
func (p *Plex) Test(ctx context.Context) error {
	return notification.Get(ctx, p.client, p.baseURL()+"/identity", p.headers())
}

// OnGrab is a no-op for Plex — Plex has no concept of queued downloads.
func (p *Plex) OnGrab(context.Context, notification.GrabMessage) error { return nil }

// OnDownload triggers library refresh.
func (p *Plex) OnDownload(ctx context.Context, _ notification.DownloadMessage) error {
	if !p.settings.UpdateLibrary {
		return nil
	}
	// Section-less refresh — Plex refreshes all TV library sections by default.
	return notification.Get(ctx, p.client,
		p.baseURL()+"/library/sections/all/refresh", p.headers())
}

// OnHealthIssue is a no-op; Plex has no notification inbox.
func (p *Plex) OnHealthIssue(context.Context, notification.HealthMessage) error { return nil }

func (p *Plex) baseURL() string {
	scheme := "http"
	if p.settings.UseSSL {
		scheme = "https"
	}
	port := p.settings.Port
	if port == 0 {
		port = 32400
	}
	return fmt.Sprintf("%s://%s:%d", scheme, url.PathEscape(p.settings.Host), port)
}

func (p *Plex) headers() map[string]string {
	return map[string]string{
		"X-Plex-Token":             p.settings.AuthToken,
		"X-Plex-Client-Identifier": "sonarr2",
		"Accept":                   "application/json",
	}
}
