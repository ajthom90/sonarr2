// Package deluge implements a downloadclient.DownloadClient for Deluge.
// Deluge exposes a JSON-RPC API at POST /json.
// Authentication is done via auth.login before any other call.
package deluge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Deluge is a downloadclient.DownloadClient for Deluge.
type Deluge struct {
	settings Settings
	client   *http.Client
	reqID    int
}

// New constructs a Deluge download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Deluge {
	if client == nil {
		client = http.DefaultClient
	}
	return &Deluge{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (d *Deluge) Implementation() string { return "Deluge" }

// DefaultName satisfies providers.Provider.
func (d *Deluge) DefaultName() string { return "Deluge" }

// Settings satisfies providers.Provider.
func (d *Deluge) Settings() any { return &d.settings }

// Protocol satisfies downloadclient.DownloadClient.
func (d *Deluge) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// Add authenticates and adds a torrent URL.
func (d *Deluge) Add(ctx context.Context, torrentURL string, title string) (string, error) {
	if err := d.auth(ctx); err != nil {
		return "", err
	}
	params := []interface{}{torrentURL, map[string]interface{}{}}
	var result string
	if err := d.call(ctx, "core.add_torrent_url", params, &result); err != nil {
		return "", err
	}
	return result, nil
}

// Items returns the current torrent list.
func (d *Deluge) Items(ctx context.Context) ([]downloadclient.Item, error) {
	if err := d.auth(ctx); err != nil {
		return nil, err
	}
	// core.get_torrents_status returns map[torrentHash]torrentStatus
	params := []interface{}{map[string]interface{}{}, []string{"name", "state", "total_size", "total_remaining", "save_path"}}
	var result map[string]torrentStatus
	if err := d.call(ctx, "core.get_torrents_status", params, &result); err != nil {
		return nil, err
	}

	items := make([]downloadclient.Item, 0, len(result))
	for hash, t := range result {
		items = append(items, downloadclient.Item{
			DownloadID: hash,
			Title:      t.Name,
			Status:     t.State,
			TotalSize:  t.TotalSize,
			Remaining:  t.TotalRemaining,
			OutputPath: t.SavePath,
		})
	}
	return items, nil
}

// Remove deletes a torrent from Deluge.
func (d *Deluge) Remove(ctx context.Context, downloadID string, deleteData bool) error {
	if err := d.auth(ctx); err != nil {
		return err
	}
	params := []interface{}{downloadID, deleteData}
	var result bool
	return d.call(ctx, "core.remove_torrent", params, &result)
}

// Status returns the download client's overall state.
func (d *Deluge) Status(_ context.Context) (downloadclient.Status, error) {
	isLocalhost := d.settings.Host == "localhost" || d.settings.Host == "127.0.0.1"
	return downloadclient.Status{IsLocalhost: isLocalhost}, nil
}

// Test verifies Deluge is reachable by calling daemon.info.
func (d *Deluge) Test(ctx context.Context) error {
	if err := d.auth(ctx); err != nil {
		return err
	}
	var result string
	return d.call(ctx, "daemon.info", []interface{}{}, &result)
}

// auth logs in to Deluge.
func (d *Deluge) auth(ctx context.Context) error {
	var result bool
	if err := d.call(ctx, "auth.login", []interface{}{d.settings.Password}, &result); err != nil {
		return err
	}
	if !result {
		return fmt.Errorf("deluge: authentication failed")
	}
	return nil
}

// call executes a Deluge JSON-RPC request.
func (d *Deluge) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	d.reqID++
	reqBody := map[string]interface{}{
		"method": method,
		"params": params,
		"id":     d.reqID,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("deluge: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.jsonURL(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("deluge: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("deluge: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deluge: json rpc returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("deluge: failed to read response: %w", err)
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("deluge: failed to parse response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("deluge: rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if result != nil && rpcResp.Result != nil {
		return json.Unmarshal(rpcResp.Result, result)
	}
	return nil
}

// jsonURL builds the JSON-RPC endpoint URL.
func (d *Deluge) jsonURL() string {
	scheme := "http"
	if d.settings.UseSsl {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/json", scheme, d.settings.Host, d.settings.Port)
}

// --- response types ---

type torrentStatus struct {
	Name           string `json:"name"`
	State          string `json:"state"`
	TotalSize      int64  `json:"total_size"`
	TotalRemaining int64  `json:"total_remaining"`
	SavePath       string `json:"save_path"`
}
