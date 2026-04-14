// Package sendgrid implements a SendGrid email notification provider.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/SendGrid/).
package sendgrid

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for SendGrid.
type Settings struct {
	APIKey      string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	FromAddress string `json:"from" form:"text" label:"From Address" required:"true"`
	Recipients  string `json:"recipients" form:"text" label:"Recipients" required:"true" placeholder:"CSV"`
}

type SendGrid struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *SendGrid { return &SendGrid{settings: s, client: client} }

func (s *SendGrid) Implementation() string { return "SendGrid" }
func (s *SendGrid) DefaultName() string    { return "SendGrid" }
func (s *SendGrid) Settings() any          { return &s.settings }

func (s *SendGrid) Test(ctx context.Context) error {
	return s.send(ctx, "sonarr2 test", "This is a test email from sonarr2.")
}

func (s *SendGrid) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return s.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (s *SendGrid) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return s.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (s *SendGrid) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return s.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (s *SendGrid) send(ctx context.Context, subject, body string) error {
	if s.settings.APIKey == "" {
		return fmt.Errorf("sendgrid: APIKey is not configured")
	}
	tos := make([]map[string]any, 0)
	for _, r := range strings.Split(s.settings.Recipients, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			tos = append(tos, map[string]any{"email": r})
		}
	}
	if len(tos) == 0 {
		return fmt.Errorf("sendgrid: no recipients")
	}
	payload := map[string]any{
		"personalizations": []map[string]any{{"to": tos}},
		"from":             map[string]any{"email": s.settings.FromAddress},
		"subject":          subject,
		"content":          []map[string]any{{"type": "text/plain", "value": body}},
	}
	return notification.PostJSONWithHeaders(ctx, s.client, "https://api.sendgrid.com/v3/mail/send",
		payload, map[string]string{"Authorization": "Bearer " + s.settings.APIKey})
}
