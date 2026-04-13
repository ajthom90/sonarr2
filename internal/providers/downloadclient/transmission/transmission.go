// Package transmission implements a downloadclient.DownloadClient for Transmission.
// Transmission uses an HTTP RPC API at POST {urlBase}rpc.
// The first request returns a 409 with an X-Transmission-Session-Id header;
// subsequent requests must include that header.
package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

const sessionIDHeader = "X-Transmission-Session-Id"

// Transmission is a downloadclient.DownloadClient for Transmission.
type Transmission struct {
	settings  Settings
	client    *http.Client
	sessionID string
}

// New constructs a Transmission download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Transmission {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transmission{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (t *Transmission) Implementation() string { return "Transmission" }

// DefaultName satisfies providers.Provider.
func (t *Transmission) DefaultName() string { return "Transmission" }

// Settings satisfies providers.Provider.
func (t *Transmission) Settings() any { return &t.settings }

// Protocol satisfies downloadclient.DownloadClient.
func (t *Transmission) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// Add adds a torrent URL to Transmission.
func (t *Transmission) Add(ctx context.Context, torrentURL string, title string) (string, error) {
	args := map[string]interface{}{"filename": torrentURL}
	var result struct {
		TorrentAdded     *torrentRef `json:"torrent-added"`
		TorrentDuplicate *torrentRef `json:"torrent-duplicate"`
	}
	if err := t.call(ctx, "torrent-add", args, &result); err != nil {
		return "", err
	}
	if result.TorrentAdded != nil {
		return fmt.Sprintf("%d", result.TorrentAdded.ID), nil
	}
	if result.TorrentDuplicate != nil {
		return fmt.Sprintf("%d", result.TorrentDuplicate.ID), nil
	}
	return "", fmt.Errorf("transmission: torrent-add returned no torrent info")
}

// Items returns the current torrent list.
func (t *Transmission) Items(ctx context.Context) ([]downloadclient.Item, error) {
	args := map[string]interface{}{
		"fields": []string{"id", "name", "status", "totalSize", "leftUntilDone", "downloadDir"},
	}
	var result struct {
		Torrents []torrentGetItem `json:"torrents"`
	}
	if err := t.call(ctx, "torrent-get", args, &result); err != nil {
		return nil, err
	}

	items := make([]downloadclient.Item, 0, len(result.Torrents))
	for _, tor := range result.Torrents {
		items = append(items, downloadclient.Item{
			DownloadID: fmt.Sprintf("%d", tor.ID),
			Title:      tor.Name,
			Status:     transmissionStatus(tor.Status),
			TotalSize:  tor.TotalSize,
			Remaining:  tor.LeftUntilDone,
			OutputPath: tor.DownloadDir,
		})
	}
	return items, nil
}

// Remove removes a torrent from Transmission.
func (t *Transmission) Remove(ctx context.Context, downloadID string, deleteData bool) error {
	var id int64
	if _, err := fmt.Sscan(downloadID, &id); err != nil {
		return fmt.Errorf("transmission: parse download ID %q: %w", downloadID, err)
	}
	args := map[string]interface{}{
		"ids":               []int64{id},
		"delete-local-data": deleteData,
	}
	return t.call(ctx, "torrent-remove", args, nil)
}

// Status returns the download client's overall state.
func (t *Transmission) Status(_ context.Context) (downloadclient.Status, error) {
	isLocalhost := t.settings.Host == "localhost" || t.settings.Host == "127.0.0.1"
	return downloadclient.Status{IsLocalhost: isLocalhost}, nil
}

// Test verifies Transmission is reachable by calling session-get.
func (t *Transmission) Test(ctx context.Context) error {
	return t.call(ctx, "session-get", map[string]interface{}{}, nil)
}

// call executes a Transmission RPC request, handling the 409 session-ID dance.
func (t *Transmission) call(ctx context.Context, method string, args interface{}, result interface{}) error {
	body, err := t.doCall(ctx, method, args)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	var rpcResp struct {
		Result    string          `json:"result"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("transmission: failed to parse response: %w", err)
	}
	if rpcResp.Result != "success" {
		return fmt.Errorf("transmission: rpc returned %q", rpcResp.Result)
	}
	if rpcResp.Arguments != nil {
		return json.Unmarshal(rpcResp.Arguments, result)
	}
	return nil
}

// doCall executes the raw HTTP request with the session-ID retry logic.
func (t *Transmission) doCall(ctx context.Context, method string, args interface{}) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{"method": method, "arguments": args})
	if err != nil {
		return nil, fmt.Errorf("transmission: failed to marshal request: %w", err)
	}

	body, sessionID, err := t.httpPost(ctx, data, t.sessionID)
	if err != nil {
		return nil, err
	}
	// If we got a new session ID (after 409), retry.
	if sessionID != "" && sessionID != t.sessionID {
		t.sessionID = sessionID
		body, _, err = t.httpPost(ctx, data, t.sessionID)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// httpPost sends a single POST and returns the body plus a new session ID if a
// 409 was received (in which case body will be nil).
func (t *Transmission) httpPost(ctx context.Context, data []byte, sessionID string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.rpcURL(), bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("transmission: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set(sessionIDHeader, sessionID)
	}
	if t.settings.Username != "" {
		req.SetBasicAuth(t.settings.Username, t.settings.Password)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("transmission: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// 409 — grab the new session ID and signal a retry.
		newID := resp.Header.Get(sessionIDHeader)
		return nil, newID, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("transmission: rpc returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("transmission: failed to read response: %w", err)
	}
	return body, "", nil
}

// rpcURL builds the RPC endpoint URL.
func (t *Transmission) rpcURL() string {
	scheme := "http"
	if t.settings.UseSsl {
		scheme = "https"
	}
	base := t.settings.UrlBase
	if base == "" {
		base = "/transmission/"
	}
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s://%s:%d%s/rpc", scheme, t.settings.Host, t.settings.Port, base)
}

// transmissionStatus maps numeric Transmission status codes to human-readable strings.
func transmissionStatus(code int) string {
	switch code {
	case 0:
		return "stopped"
	case 1, 2:
		return "check_waiting"
	case 3, 4:
		return "downloading"
	case 5, 6:
		return "seeding"
	default:
		return "unknown"
	}
}

// --- response types ---

type torrentRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type torrentGetItem struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Status        int    `json:"status"`
	TotalSize     int64  `json:"totalSize"`
	LeftUntilDone int64  `json:"leftUntilDone"`
	DownloadDir   string `json:"downloadDir"`
}
