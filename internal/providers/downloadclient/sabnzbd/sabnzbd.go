// Package sabnzbd implements a downloadclient.DownloadClient for SABnzbd.
package sabnzbd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// SABnzbd is a downloadclient.DownloadClient that talks to SABnzbd's JSON API.
type SABnzbd struct {
	settings Settings
	client   *http.Client
}

// New constructs a SABnzbd download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *SABnzbd {
	if client == nil {
		client = http.DefaultClient
	}
	return &SABnzbd{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (s *SABnzbd) Implementation() string { return "SABnzbd" }

// DefaultName satisfies providers.Provider.
func (s *SABnzbd) DefaultName() string { return "SABnzbd" }

// Settings satisfies providers.Provider.
func (s *SABnzbd) Settings() any { return &s.settings }

// Protocol satisfies downloadclient.DownloadClient.
func (s *SABnzbd) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }

// Add submits a URL to SABnzbd for download and returns the NZO ID.
// GET {baseURL}/api?mode=addurl&name={url}&nzbname={title}&cat={category}&apikey={key}&output=json
func (s *SABnzbd) Add(ctx context.Context, nzbURL string, title string) (string, error) {
	u, err := url.Parse(s.baseURL() + "/api")
	if err != nil {
		return "", fmt.Errorf("sabnzbd: failed to parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("mode", "addurl")
	q.Set("name", nzbURL)
	q.Set("nzbname", title)
	q.Set("cat", s.settings.Category)
	q.Set("apikey", s.settings.ApiKey)
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	body, err := s.get(ctx, u.String())
	if err != nil {
		return "", err
	}

	var resp addResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("sabnzbd: failed to parse add response: %w", err)
	}
	if !resp.Status {
		return "", fmt.Errorf("sabnzbd: add returned status=false")
	}
	if len(resp.NzoIDs) == 0 {
		return "", fmt.Errorf("sabnzbd: add returned no nzo_ids")
	}
	return resp.NzoIDs[0], nil
}

// Items returns the current download queue as a slice of downloadclient.Item.
// GET {baseURL}/api?mode=queue&apikey={key}&output=json
func (s *SABnzbd) Items(ctx context.Context) ([]downloadclient.Item, error) {
	u, err := url.Parse(s.baseURL() + "/api")
	if err != nil {
		return nil, fmt.Errorf("sabnzbd: failed to parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("mode", "queue")
	q.Set("apikey", s.settings.ApiKey)
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	body, err := s.get(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var resp queueResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("sabnzbd: failed to parse queue response: %w", err)
	}

	items := make([]downloadclient.Item, 0, len(resp.Queue.Slots))
	for _, slot := range resp.Queue.Slots {
		totalMB, _ := strconv.ParseFloat(slot.MB, 64)
		leftMB, _ := strconv.ParseFloat(slot.MBLeft, 64)

		items = append(items, downloadclient.Item{
			DownloadID: slot.NzoID,
			Title:      slot.Filename,
			Status:     slot.Status,
			TotalSize:  int64(totalMB * 1024 * 1024),
			Remaining:  int64(leftMB * 1024 * 1024),
			OutputPath: slot.Storage,
		})
	}
	return items, nil
}

// Remove deletes the item identified by downloadID from the SABnzbd queue.
// GET {baseURL}/api?mode=queue&name=delete&value={id}&del_files={0|1}&apikey={key}&output=json
func (s *SABnzbd) Remove(ctx context.Context, downloadID string, deleteData bool) error {
	u, err := url.Parse(s.baseURL() + "/api")
	if err != nil {
		return fmt.Errorf("sabnzbd: failed to parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("mode", "queue")
	q.Set("name", "delete")
	q.Set("value", downloadID)
	delFiles := "0"
	if deleteData {
		delFiles = "1"
	}
	q.Set("del_files", delFiles)
	q.Set("apikey", s.settings.ApiKey)
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	_, err = s.get(ctx, u.String())
	return err
}

// Status returns the current state of the SABnzbd download client.
func (s *SABnzbd) Status(_ context.Context) (downloadclient.Status, error) {
	isLocalhost := s.settings.Host == "localhost" || s.settings.Host == "127.0.0.1"
	return downloadclient.Status{
		IsLocalhost:       isLocalhost,
		OutputRootFolders: nil,
	}, nil
}

// Test verifies that SABnzbd is reachable and the API key is accepted.
// GET {baseURL}/api?mode=version&apikey={key}&output=json
func (s *SABnzbd) Test(ctx context.Context) error {
	u, err := url.Parse(s.baseURL() + "/api")
	if err != nil {
		return fmt.Errorf("sabnzbd: failed to parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("mode", "version")
	q.Set("apikey", s.settings.ApiKey)
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sabnzbd: version returned status %d", resp.StatusCode)
	}
	return nil
}

// baseURL constructs the scheme+host+port base URL from settings.
func (s *SABnzbd) baseURL() string {
	scheme := "http"
	if s.settings.UseSsl {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, s.settings.Host, s.settings.Port)
}

// get executes an HTTP GET and returns the response body, failing on non-200 status.
func (s *SABnzbd) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sabnzbd: request to %s returned status %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// --- JSON response types ---

// addResponse is the JSON response from mode=addurl.
type addResponse struct {
	Status bool     `json:"status"`
	NzoIDs []string `json:"nzo_ids"`
}

// queueResponse is the JSON response from mode=queue.
type queueResponse struct {
	Queue queueBody `json:"queue"`
}

type queueBody struct {
	Slots []queueSlot `json:"slots"`
}

type queueSlot struct {
	NzoID    string `json:"nzo_id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	MB       string `json:"mb"`
	MBLeft   string `json:"mbleft"`
	Storage  string `json:"storage"`
}
