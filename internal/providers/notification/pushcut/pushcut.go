// Package pushcut implements Pushcut (pushcut.io) iOS push notifications.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Pushcut/).
package pushcut

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for Pushcut.
type Settings struct {
	APIKey           string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	NotificationName string `json:"notificationName" form:"text" label:"Notification Name" required:"true"`
	Devices          string `json:"devices" form:"text" label:"Devices"`
}

type Pushcut struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Pushcut { return &Pushcut{settings: s, client: client} }

func (p *Pushcut) Implementation() string { return "Pushcut" }
func (p *Pushcut) DefaultName() string    { return "Pushcut" }
func (p *Pushcut) Settings() any          { return &p.settings }

func (p *Pushcut) Test(ctx context.Context) error {
	return p.send(ctx, "Test", "sonarr2 notification test")
}

func (p *Pushcut) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return p.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Pushcut) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return p.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Pushcut) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return p.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (p *Pushcut) send(ctx context.Context, title, body string) error {
	if p.settings.APIKey == "" || p.settings.NotificationName == "" {
		return fmt.Errorf("pushcut: APIKey and NotificationName are required")
	}
	payload := map[string]any{
		"title": title,
		"text":  body,
	}
	if p.settings.Devices != "" {
		// Devices is a comma-separated list per Pushcut.
		payload["devices"] = p.settings.Devices
	}
	endpoint := "https://api.pushcut.io/v1/notifications/" + url.PathEscape(p.settings.NotificationName)
	return notification.PostJSONWithHeaders(ctx, p.client, endpoint, payload,
		map[string]string{"API-Key": p.settings.APIKey})
}
