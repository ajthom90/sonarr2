// Package prowl implements Prowl iOS push notifications.
// API docs: https://www.prowlapp.com/api.php
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Prowl/).
package prowl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for a Prowl notification provider.
type Settings struct {
	APIKey   string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	Priority int    `json:"priority" form:"number" label:"Priority" placeholder:"0"` // -2..2
}

// Prowl sends push notifications to Prowl-connected iOS devices.
type Prowl struct {
	settings Settings
	client   *http.Client
}

// New constructs a Prowl provider.
func New(s Settings, client *http.Client) *Prowl { return &Prowl{settings: s, client: client} }

func (p *Prowl) Implementation() string { return "Prowl" }
func (p *Prowl) DefaultName() string    { return "Prowl" }
func (p *Prowl) Settings() any          { return &p.settings }

func (p *Prowl) Test(ctx context.Context) error {
	return p.push(ctx, "Test", "sonarr2 notification test")
}

func (p *Prowl) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return p.push(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Prowl) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return p.push(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (p *Prowl) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return p.push(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (p *Prowl) push(ctx context.Context, event, desc string) error {
	if p.settings.APIKey == "" {
		return fmt.Errorf("prowl: APIKey is not configured")
	}
	v := url.Values{}
	v.Set("apikey", p.settings.APIKey)
	v.Set("application", "Sonarr")
	v.Set("event", event)
	v.Set("description", desc)
	if p.settings.Priority != 0 {
		v.Set("priority", strconv.Itoa(p.settings.Priority))
	}
	return notification.PostForm(ctx, p.client, "https://api.prowlapp.com/publicapi/add", v.Encode(), nil)
}
