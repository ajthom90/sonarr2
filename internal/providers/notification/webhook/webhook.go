// Package webhook implements a notification.Notification that POSTs (or GETs)
// a JSON payload to a configurable URL.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Webhook sends notifications to a generic HTTP endpoint.
type Webhook struct {
	settings Settings
	client   *http.Client
}

// New constructs a Webhook notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Webhook {
	if client == nil {
		client = http.DefaultClient
	}
	return &Webhook{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (w *Webhook) Implementation() string { return "Webhook" }

// DefaultName satisfies providers.Provider.
func (w *Webhook) DefaultName() string { return "Webhook" }

// Settings satisfies providers.Provider.
func (w *Webhook) Settings() any { return &w.settings }

// Test verifies the webhook URL is configured.
func (w *Webhook) Test(ctx context.Context) error {
	if w.settings.URL == "" {
		return fmt.Errorf("webhook: URL is not configured")
	}
	payload := map[string]any{"eventType": "Test"}
	return w.post(ctx, payload)
}

// OnGrab sends a grab event to the configured webhook.
func (w *Webhook) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	payload := map[string]any{
		"eventType":    "Grab",
		"seriesTitle":  msg.SeriesTitle,
		"episodeTitle": msg.EpisodeTitle,
		"quality":      msg.Quality,
		"indexer":      msg.Indexer,
	}
	return w.post(ctx, payload)
}

// OnDownload sends a download event to the configured webhook.
func (w *Webhook) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	payload := map[string]any{
		"eventType":    "Download",
		"seriesTitle":  msg.SeriesTitle,
		"episodeTitle": msg.EpisodeTitle,
		"quality":      msg.Quality,
	}
	return w.post(ctx, payload)
}

// OnHealthIssue sends a health issue event to the configured webhook.
func (w *Webhook) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	payload := map[string]any{
		"eventType": "HealthIssue",
		"type":      msg.Type,
		"message":   msg.Message,
	}
	return w.post(ctx, payload)
}

// post sends a JSON payload to the configured URL using the configured method.
func (w *Webhook) post(ctx context.Context, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	method := strings.ToUpper(w.settings.Method)
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, w.settings.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: request returned status %d", resp.StatusCode)
	}
	return nil
}
