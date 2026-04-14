// Package pushbullet implements a notification.Notification that posts to the
// Pushbullet API (https://api.pushbullet.com/v2/pushes).
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/PushBullet/).
package pushbullet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

const apiURL = "https://api.pushbullet.com/v2/pushes"

// Pushbullet sends notifications to Pushbullet-connected devices.
type Pushbullet struct {
	settings Settings
	client   *http.Client
}

// New constructs a Pushbullet provider.
func New(settings Settings, client *http.Client) *Pushbullet {
	if client == nil {
		client = http.DefaultClient
	}
	return &Pushbullet{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (p *Pushbullet) Implementation() string { return "PushBullet" }

// DefaultName satisfies providers.Provider.
func (p *Pushbullet) DefaultName() string { return "Pushbullet" }

// Settings satisfies providers.Provider.
func (p *Pushbullet) Settings() any { return &p.settings }

// Test sends a test notification.
func (p *Pushbullet) Test(ctx context.Context) error {
	if p.settings.APIKey == "" {
		return fmt.Errorf("pushbullet: APIKey is not configured")
	}
	return p.push(ctx, "Test", "sonarr2 notification test")
}

// OnGrab sends a "Release Grabbed" push.
func (p *Pushbullet) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return p.push(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

// OnDownload sends a "Download Complete" push.
func (p *Pushbullet) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return p.push(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

// OnHealthIssue sends a health warning push.
func (p *Pushbullet) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return p.push(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

// push dispatches one note push, optionally targeting specific device IDs or channel tags.
func (p *Pushbullet) push(ctx context.Context, title, body string) error {
	// Sonarr sends one push per device_iden and per channel_tag; if both are empty
	// a single push is sent with no targeting (broadcasts to all devices).
	devs := splitCSV(p.settings.DeviceIds)
	chans := splitCSV(p.settings.ChannelTags)
	if len(devs) == 0 && len(chans) == 0 {
		return p.one(ctx, map[string]any{"type": "note", "title": title, "body": body})
	}
	for _, dev := range devs {
		if err := p.one(ctx, map[string]any{
			"type": "note", "title": title, "body": body,
			"device_iden": dev, "source_device_iden": p.settings.SenderID,
		}); err != nil {
			return err
		}
	}
	for _, ch := range chans {
		if err := p.one(ctx, map[string]any{
			"type": "note", "title": title, "body": body,
			"channel_tag": ch,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pushbullet) one(ctx context.Context, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("pushbullet: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pushbullet: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", p.settings.APIKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pushbullet: post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	buf, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("pushbullet: status %d: %s", resp.StatusCode, strings.TrimSpace(string(buf)))
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
