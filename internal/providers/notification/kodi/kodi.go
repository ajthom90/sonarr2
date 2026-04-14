// Package kodi implements a notification.Notification for Kodi/XBMC via
// JSON-RPC. Supports GUI on-screen notifications, VideoLibrary.Scan, and
// VideoLibrary.Clean.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Xbmc/).
package kodi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Kodi represents a single Kodi/XBMC media player instance.
type Kodi struct {
	settings Settings
	client   *http.Client
}

// New constructs a Kodi provider.
func New(settings Settings, client *http.Client) *Kodi {
	if client == nil {
		client = http.DefaultClient
	}
	return &Kodi{settings: settings, client: client}
}

func (k *Kodi) Implementation() string { return "Xbmc" }
func (k *Kodi) DefaultName() string    { return "Kodi (XBMC)" }
func (k *Kodi) Settings() any          { return &k.settings }

func (k *Kodi) Test(ctx context.Context) error {
	return k.notify(ctx, "Test", "sonarr2 notification test")
}

func (k *Kodi) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	if !k.settings.Notify {
		return nil
	}
	return k.notify(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (k *Kodi) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	if k.settings.Notify {
		if err := k.notify(ctx, "Sonarr - Download Complete",
			fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle)); err != nil {
			return err
		}
	}
	if k.settings.UpdateLibrary {
		if err := k.call(ctx, "VideoLibrary.Scan", nil); err != nil {
			return fmt.Errorf("kodi: scan library: %w", err)
		}
	}
	if k.settings.CleanLibrary {
		if err := k.call(ctx, "VideoLibrary.Clean", map[string]any{"showdialogs": false}); err != nil {
			return fmt.Errorf("kodi: clean library: %w", err)
		}
	}
	return nil
}

func (k *Kodi) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	if !k.settings.Notify {
		return nil
	}
	return k.notify(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (k *Kodi) notify(ctx context.Context, title, body string) error {
	display := k.settings.DisplayTime * 1000
	if display <= 0 {
		display = 5000
	}
	return k.call(ctx, "GUI.ShowNotification", map[string]any{
		"title":       title,
		"message":     body,
		"displaytime": display,
	})
}

func (k *Kodi) call(ctx context.Context, method string, params map[string]any) error {
	scheme := "http"
	if k.settings.UseSSL {
		scheme = "https"
	}
	port := k.settings.Port
	if port == 0 {
		port = 8080
	}
	url := fmt.Sprintf("%s://%s:%d/jsonrpc", scheme, k.settings.Host, port)

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("kodi: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if k.settings.Username != "" && k.settings.Password != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(k.settings.Username + ":" + k.settings.Password))
		req.Header.Set("Authorization", "Basic "+creds)
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("kodi: call %s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("kodi: %s status %d: %s", method, resp.StatusCode, strings.TrimSpace(string(buf)))
	}
	return nil
}
