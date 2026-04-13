// Package discord implements a notification.Notification that POSTs Discord embeds
// to a configured webhook URL.
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Discord sends notifications to a Discord channel via a webhook URL.
type Discord struct {
	settings Settings
	client   *http.Client
}

// New constructs a Discord notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Discord {
	if client == nil {
		client = http.DefaultClient
	}
	return &Discord{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (d *Discord) Implementation() string { return "Discord" }

// DefaultName satisfies providers.Provider.
func (d *Discord) DefaultName() string { return "Discord" }

// Settings satisfies providers.Provider.
func (d *Discord) Settings() any { return &d.settings }

// Test verifies the webhook URL is set and reachable.
func (d *Discord) Test(ctx context.Context) error {
	if d.settings.WebhookURL == "" {
		return fmt.Errorf("discord: WebhookURL is not configured")
	}
	return d.sendEmbed(ctx, "Test", "sonarr2 notification test", 3447003)
}

// OnGrab sends a "Release Grabbed" embed to Discord.
func (d *Discord) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	description := fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle)
	return d.sendEmbed(ctx, "Release Grabbed", description, 3066993)
}

// OnDownload sends a "Download Complete" embed to Discord.
func (d *Discord) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	description := fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle)
	return d.sendEmbed(ctx, "Download Complete", description, 3066993)
}

// OnHealthIssue sends a "Health Issue" embed to Discord.
func (d *Discord) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	description := fmt.Sprintf("[%s] %s", msg.Type, msg.Message)
	return d.sendEmbed(ctx, "Health Issue", description, 15158332)
}

// sendEmbed POSTs a Discord embed payload to the configured webhook URL.
func (d *Discord) sendEmbed(ctx context.Context, title, description string, color int) error {
	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": description,
				"color":       color,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.settings.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: webhook returned status %d", resp.StatusCode)
	}
	return nil
}
