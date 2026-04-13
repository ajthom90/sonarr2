// Package qbittorrent implements a downloadclient.DownloadClient for qBittorrent.
// It uses the qBittorrent Web API v2.
package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// QBittorrent is a downloadclient.DownloadClient for qBittorrent.
type QBittorrent struct {
	settings Settings
	client   *http.Client
}

// New constructs a QBittorrent download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *QBittorrent {
	if client == nil {
		client = http.DefaultClient
	}
	return &QBittorrent{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (q *QBittorrent) Implementation() string { return "qBittorrent" }

// DefaultName satisfies providers.Provider.
func (q *QBittorrent) DefaultName() string { return "qBittorrent" }

// Settings satisfies providers.Provider.
func (q *QBittorrent) Settings() any { return &q.settings }

// Protocol satisfies downloadclient.DownloadClient.
func (q *QBittorrent) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// Add logs in to qBittorrent and adds the torrent URL to the queue.
// Returns the URL as a pseudo download-ID (qBittorrent uses info-hash, which
// we can't know without parsing the torrent file).
func (q *QBittorrent) Add(ctx context.Context, torrentURL string, title string) (string, error) {
	if err := q.login(ctx); err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("urls", torrentURL)
	if q.settings.Category != "" {
		form.Set("category", q.settings.Category)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		q.baseURL()+"/api/v2/torrents/add",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("qbittorrent: failed to build add request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("qbittorrent: add request failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("qbittorrent: add returned status %d", resp.StatusCode)
	}
	return torrentURL, nil
}

// Items returns the torrent list for the configured category.
func (q *QBittorrent) Items(ctx context.Context) ([]downloadclient.Item, error) {
	if err := q.login(ctx); err != nil {
		return nil, err
	}

	u := q.baseURL() + "/api/v2/torrents/info"
	if q.settings.Category != "" {
		u += "?category=" + url.QueryEscape(q.settings.Category)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: failed to build items request: %w", err)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: items request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbittorrent: torrents/info returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: failed to read items response: %w", err)
	}

	var torrents []torrentInfo
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, fmt.Errorf("qbittorrent: failed to parse items response: %w", err)
	}

	items := make([]downloadclient.Item, 0, len(torrents))
	for _, t := range torrents {
		items = append(items, downloadclient.Item{
			DownloadID: t.Hash,
			Title:      t.Name,
			Status:     t.State,
			TotalSize:  t.Size,
			Remaining:  t.AmountLeft,
			OutputPath: t.SavePath,
		})
	}
	return items, nil
}

// Remove deletes a torrent (and optionally its data) from qBittorrent.
func (q *QBittorrent) Remove(ctx context.Context, downloadID string, deleteData bool) error {
	if err := q.login(ctx); err != nil {
		return err
	}

	deleteFiles := "false"
	if deleteData {
		deleteFiles = "true"
	}
	form := url.Values{}
	form.Set("hashes", downloadID)
	form.Set("deleteFiles", deleteFiles)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		q.baseURL()+"/api/v2/torrents/delete",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("qbittorrent: failed to build remove request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent: remove request failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent: delete returned status %d", resp.StatusCode)
	}
	return nil
}

// Status returns the download client's overall state.
func (q *QBittorrent) Status(_ context.Context) (downloadclient.Status, error) {
	isLocalhost := q.settings.Host == "localhost" || q.settings.Host == "127.0.0.1"
	return downloadclient.Status{IsLocalhost: isLocalhost}, nil
}

// Test verifies qBittorrent is reachable by fetching the application version.
func (q *QBittorrent) Test(ctx context.Context) error {
	if err := q.login(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q.baseURL()+"/api/v2/app/version", nil)
	if err != nil {
		return fmt.Errorf("qbittorrent: failed to build version request: %w", err)
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent: version request failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent: version returned status %d", resp.StatusCode)
	}
	return nil
}

// login authenticates to qBittorrent and obtains a session cookie.
// The http.Client's cookie jar (if set) handles the session automatically.
func (q *QBittorrent) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", q.settings.Username)
	form.Set("password", q.settings.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		q.baseURL()+"/api/v2/auth/login",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("qbittorrent: failed to build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent: login request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent: login returned status %d", resp.StatusCode)
	}
	if strings.TrimSpace(string(body)) == "Fails." {
		return fmt.Errorf("qbittorrent: login failed — bad credentials")
	}
	return nil
}

// baseURL builds the base URL from settings.
func (q *QBittorrent) baseURL() string {
	scheme := "http"
	if q.settings.UseSsl {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, q.settings.Host, q.settings.Port)
}

// --- response types ---

type torrentInfo struct {
	Hash       string `json:"hash"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Size       int64  `json:"size"`
	AmountLeft int64  `json:"amount_left"`
	SavePath   string `json:"save_path"`
}
