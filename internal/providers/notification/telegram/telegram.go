// Package telegram implements a notification.Notification that sends messages
// via the Telegram Bot API.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

const telegramAPIBase = "https://api.telegram.org"

// Telegram sends notifications via the Telegram Bot API.
type Telegram struct {
	settings Settings
	client   *http.Client
	apiBase  string // overridable in tests
}

// New constructs a Telegram notification provider. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Telegram {
	if client == nil {
		client = http.DefaultClient
	}
	return &Telegram{settings: settings, client: client, apiBase: telegramAPIBase}
}

// Implementation satisfies providers.Provider.
func (t *Telegram) Implementation() string { return "Telegram" }

// DefaultName satisfies providers.Provider.
func (t *Telegram) DefaultName() string { return "Telegram" }

// Settings satisfies providers.Provider.
func (t *Telegram) Settings() any { return &t.settings }

// Test verifies the bot token and chat ID are configured and reachable.
func (t *Telegram) Test(ctx context.Context) error {
	if t.settings.BotToken == "" {
		return fmt.Errorf("telegram: BotToken is not configured")
	}
	return t.sendMessage(ctx, "sonarr2 notification test")
}

// OnGrab sends a "Release Grabbed" message.
func (t *Telegram) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	text := fmt.Sprintf("Release Grabbed: %s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return t.sendMessage(ctx, text)
}

// OnDownload sends a "Download Complete" message.
func (t *Telegram) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	text := fmt.Sprintf("Download Complete: %s — %s [%s]", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return t.sendMessage(ctx, text)
}

// OnHealthIssue sends a "Health Issue" message.
func (t *Telegram) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	text := fmt.Sprintf("Health Issue [%s]: %s", msg.Type, msg.Message)
	return t.sendMessage(ctx, text)
}

// sendMessage POSTs a sendMessage request to the Telegram Bot API.
func (t *Telegram) sendMessage(ctx context.Context, text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.settings.BotToken)
	payload := map[string]string{
		"chat_id": t.settings.ChatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: post message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: sendMessage returned status %d", resp.StatusCode)
	}
	return nil
}
