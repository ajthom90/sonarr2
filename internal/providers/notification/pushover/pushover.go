// Package pushover implements a notification.Notification that delivers push
// notifications via the Pushover API.
package pushover

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

// Pushover sends notifications via the Pushover API.
type Pushover struct {
	settings Settings
	client   *http.Client
	apiURL   string // overridable in tests
}

// New constructs a Pushover notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Pushover {
	if client == nil {
		client = http.DefaultClient
	}
	return &Pushover{settings: settings, client: client, apiURL: pushoverAPIURL}
}

// Implementation satisfies providers.Provider.
func (p *Pushover) Implementation() string { return "Pushover" }

// DefaultName satisfies providers.Provider.
func (p *Pushover) DefaultName() string { return "Pushover" }

// Settings satisfies providers.Provider.
func (p *Pushover) Settings() any { return &p.settings }

// Test verifies that the Pushover credentials are configured.
func (p *Pushover) Test(ctx context.Context) error {
	if p.settings.UserKey == "" {
		return fmt.Errorf("pushover: UserKey is not configured")
	}
	if p.settings.ApiToken == "" {
		return fmt.Errorf("pushover: ApiToken is not configured")
	}
	return p.sendMessage(ctx, "Test", "sonarr2 notification test")
}

// OnGrab sends a "Release Grabbed" push notification.
func (p *Pushover) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	title := "Release Grabbed"
	message := fmt.Sprintf("%s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return p.sendMessage(ctx, title, message)
}

// OnDownload sends a "Download Complete" push notification.
func (p *Pushover) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	title := "Download Complete"
	message := fmt.Sprintf("%s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return p.sendMessage(ctx, title, message)
}

// OnHealthIssue sends a "Health Issue" push notification.
func (p *Pushover) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	title := "Health Issue"
	message := fmt.Sprintf("[%s] %s", msg.Type, msg.Message)
	return p.sendMessage(ctx, title, message)
}

// sendMessage POSTs to the Pushover messages API with form data.
func (p *Pushover) sendMessage(ctx context.Context, title, message string) error {
	form := url.Values{}
	form.Set("token", p.settings.ApiToken)
	form.Set("user", p.settings.UserKey)
	form.Set("title", title)
	form.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, nil)
	if err != nil {
		return fmt.Errorf("pushover: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.ContentLength = int64(len(form.Encode()))

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushover: API returned status %d", resp.StatusCode)
	}
	return nil
}
