// Package simplepush implements Simplepush.io push notifications.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Simplepush/).
package simplepush

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for Simplepush.
type Settings struct {
	Key   string `json:"key" form:"text" label:"Key" required:"true" privacy:"apiKey"`
	Event string `json:"event" form:"text" label:"Event"`
}

type Simplepush struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Simplepush {
	return &Simplepush{settings: s, client: client}
}

func (p *Simplepush) Implementation() string { return "Simplepush" }
func (p *Simplepush) DefaultName() string    { return "Simplepush" }
func (p *Simplepush) Settings() any          { return &p.settings }

func (p *Simplepush) Test(ctx context.Context) error {
	return p.send(ctx, "Test", "sonarr2 notification test")
}

func (p *Simplepush) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return p.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Simplepush) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return p.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Simplepush) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return p.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (p *Simplepush) send(ctx context.Context, title, msg string) error {
	if p.settings.Key == "" {
		return fmt.Errorf("simplepush: Key is not configured")
	}
	v := url.Values{}
	v.Set("key", p.settings.Key)
	v.Set("title", title)
	v.Set("msg", msg)
	if p.settings.Event != "" {
		v.Set("event", p.settings.Event)
	}
	return notification.PostForm(ctx, p.client, "https://api.simplepush.io/send", v.Encode(), nil)
}
