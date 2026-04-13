// Package slack implements a notification.Notification that POSTs Slack
// attachment payloads to a configured incoming webhook URL.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Slack sends notifications to a Slack channel via an incoming webhook.
type Slack struct {
	settings Settings
	client   *http.Client
}

// New constructs a Slack notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Slack {
	if client == nil {
		client = http.DefaultClient
	}
	return &Slack{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (s *Slack) Implementation() string { return "Slack" }

// DefaultName satisfies providers.Provider.
func (s *Slack) DefaultName() string { return "Slack" }

// Settings satisfies providers.Provider.
func (s *Slack) Settings() any { return &s.settings }

// Test verifies the webhook URL is set and reachable.
func (s *Slack) Test(ctx context.Context) error {
	if s.settings.WebhookURL == "" {
		return fmt.Errorf("slack: WebhookURL is not configured")
	}
	return s.sendAttachment(ctx, "sonarr2 notification test", "good")
}

// OnGrab sends a "Release Grabbed" attachment to Slack.
func (s *Slack) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	text := fmt.Sprintf("Release Grabbed: *%s* — %s", msg.SeriesTitle, msg.EpisodeTitle)
	return s.sendAttachment(ctx, text, "good")
}

// OnDownload sends a "Download Complete" attachment to Slack.
func (s *Slack) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	text := fmt.Sprintf("Download Complete: *%s* — %s", msg.SeriesTitle, msg.EpisodeTitle)
	return s.sendAttachment(ctx, text, "good")
}

// OnHealthIssue sends a "Health Issue" attachment to Slack.
func (s *Slack) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	text := fmt.Sprintf("Health Issue [%s]: %s", msg.Type, msg.Message)
	return s.sendAttachment(ctx, text, "danger")
}

// sendAttachment POSTs a Slack webhook payload with a single attachment.
func (s *Slack) sendAttachment(ctx context.Context, text, color string) error {
	payload := map[string]any{
		"channel": s.settings.Channel,
		"attachments": []map[string]any{
			{
				"text":  text,
				"color": color,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.settings.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: webhook returned status %d", resp.StatusCode)
	}
	return nil
}
