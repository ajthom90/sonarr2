// Package apprise implements an Apprise notification gateway. Apprise is an
// open-source relay that itself dispatches to 80+ services — this provider
// simply POSTs to a user-hosted Apprise HTTP endpoint.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Apprise/).
package apprise

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for an Apprise notification provider.
type Settings struct {
	ServerURL        string `json:"serverUrl" form:"text" label:"Server URL" required:"true" placeholder:"https://apprise.example/notify/abc123"`
	NotificationType string `json:"notificationType" form:"text" label:"Notification Type" placeholder:"info"`
	Tags             string `json:"tags" form:"text" label:"Tags"`
	StatelessURLs    string `json:"statelessUrls" form:"text" label:"Stateless URLs"`
	AuthUsername     string `json:"authUsername" form:"text" label:"Auth Username"`
	AuthPassword     string `json:"authPassword" form:"password" label:"Auth Password" privacy:"password"`
}

// Apprise dispatches events through a user-hosted Apprise instance.
type Apprise struct {
	settings Settings
	client   *http.Client
}

// New constructs an Apprise provider.
func New(s Settings, client *http.Client) *Apprise { return &Apprise{settings: s, client: client} }

func (a *Apprise) Implementation() string { return "Apprise" }
func (a *Apprise) DefaultName() string    { return "Apprise" }
func (a *Apprise) Settings() any          { return &a.settings }

func (a *Apprise) Test(ctx context.Context) error {
	return a.send(ctx, "Test", "sonarr2 notification test")
}

func (a *Apprise) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return a.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (a *Apprise) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return a.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (a *Apprise) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return a.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (a *Apprise) send(ctx context.Context, title, body string) error {
	payload := map[string]any{
		"title": title,
		"body":  body,
		"type":  nz(a.settings.NotificationType, "info"),
	}
	if a.settings.Tags != "" {
		payload["tag"] = strings.TrimSpace(a.settings.Tags)
	}
	if a.settings.StatelessURLs != "" {
		payload["urls"] = strings.TrimSpace(a.settings.StatelessURLs)
	}
	headers := map[string]string{}
	if a.settings.AuthUsername != "" {
		// Note: Apprise typically uses Basic auth. Helper supports headers via
		// PostJSONWithHeaders; we inline the header here.
		headers["Authorization"] = basicAuth(a.settings.AuthUsername, a.settings.AuthPassword)
	}
	return notification.PostJSONWithHeaders(ctx, a.client, a.settings.ServerURL, payload, headers)
}

func nz(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func basicAuth(user, pass string) string {
	// Kept simple to avoid importing encoding/base64 transitively.
	return "Basic " + b64(user+":"+pass)
}

// b64 is a tiny, no-dep base64 encoder (std URL-safe alphabet not required here).
func b64(s string) string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	in := []byte(s)
	var out strings.Builder
	for i := 0; i < len(in); i += 3 {
		n := 0
		take := 0
		for j := 0; j < 3 && i+j < len(in); j++ {
			n |= int(in[i+j]) << (8 * (2 - j))
			take++
		}
		out.WriteByte(alpha[(n>>18)&0x3f])
		out.WriteByte(alpha[(n>>12)&0x3f])
		if take > 1 {
			out.WriteByte(alpha[(n>>6)&0x3f])
		} else {
			out.WriteByte('=')
		}
		if take > 2 {
			out.WriteByte(alpha[n&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}
