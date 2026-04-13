// Package gotify implements a notification.Notification that sends push
// notifications to a self-hosted Gotify server.
package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Gotify sends notifications to a Gotify server.
type Gotify struct {
	settings Settings
	client   *http.Client
}

// New constructs a Gotify notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Gotify {
	if client == nil {
		client = http.DefaultClient
	}
	return &Gotify{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (g *Gotify) Implementation() string { return "Gotify" }

// DefaultName satisfies providers.Provider.
func (g *Gotify) DefaultName() string { return "Gotify" }

// Settings satisfies providers.Provider.
func (g *Gotify) Settings() any { return &g.settings }

// Test verifies the Gotify server and app token are configured.
func (g *Gotify) Test(ctx context.Context) error {
	if g.settings.ServerURL == "" {
		return fmt.Errorf("gotify: ServerURL is not configured")
	}
	if g.settings.AppToken == "" {
		return fmt.Errorf("gotify: AppToken is not configured")
	}
	return g.sendMessage(ctx, "Test", "sonarr2 notification test")
}

// OnGrab sends a "Release Grabbed" message.
func (g *Gotify) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	title := "Release Grabbed"
	message := fmt.Sprintf("%s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return g.sendMessage(ctx, title, message)
}

// OnDownload sends a "Download Complete" message.
func (g *Gotify) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	title := "Download Complete"
	message := fmt.Sprintf("%s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return g.sendMessage(ctx, title, message)
}

// OnHealthIssue sends a "Health Issue" message.
func (g *Gotify) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	title := "Health Issue"
	message := fmt.Sprintf("[%s] %s", msg.Type, msg.Message)
	return g.sendMessage(ctx, title, message)
}

// sendMessage POSTs a message to the Gotify /message endpoint.
func (g *Gotify) sendMessage(ctx context.Context, title, message string) error {
	serverURL := strings.TrimRight(g.settings.ServerURL, "/")
	url := serverURL + "/message"

	payload := map[string]any{
		"title":    title,
		"message":  message,
		"priority": 5,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gotify: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gotify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", g.settings.AppToken)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("gotify: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify: /message returned status %d", resp.StatusCode)
	}
	return nil
}
