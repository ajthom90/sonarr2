// Package join implements Join (joaoapps.com/join) push notifications.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Join/).
package join

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for a Join notification provider.
type Settings struct {
	APIKey    string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	DeviceIds string `json:"deviceIds" form:"text" label:"Device IDs" placeholder:"Comma-separated or group name"`
	Priority  int    `json:"priority" form:"number" label:"Priority" placeholder:"2"` // -2..2
}

// Join sends push notifications through the Join joaoapps service.
type Join struct {
	settings Settings
	client   *http.Client
}

// New constructs a Join provider.
func New(s Settings, client *http.Client) *Join { return &Join{settings: s, client: client} }

func (j *Join) Implementation() string { return "Join" }
func (j *Join) DefaultName() string    { return "Join" }
func (j *Join) Settings() any          { return &j.settings }

func (j *Join) Test(ctx context.Context) error {
	return j.send(ctx, "Test", "sonarr2 notification test")
}

func (j *Join) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return j.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (j *Join) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return j.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (j *Join) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return j.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (j *Join) send(ctx context.Context, title, text string) error {
	if j.settings.APIKey == "" {
		return fmt.Errorf("join: APIKey is not configured")
	}
	v := url.Values{}
	v.Set("apikey", j.settings.APIKey)
	v.Set("title", title)
	v.Set("text", text)
	if strings.TrimSpace(j.settings.DeviceIds) != "" {
		v.Set("deviceIds", strings.TrimSpace(j.settings.DeviceIds))
	}
	v.Set("priority", fmt.Sprintf("%d", j.settings.Priority))
	return notification.Get(ctx, j.client,
		"https://joinjoaomgcd.appspot.com/_ah/api/messaging/v1/sendPush?"+v.Encode(), nil)
}
