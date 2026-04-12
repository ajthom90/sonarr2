// Package newznab implements an indexer.Indexer for Newznab-compatible APIs.
package newznab

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Newznab is an indexer.Indexer that talks to Newznab-compatible APIs.
type Newznab struct {
	settings Settings
	client   *http.Client
}

// New constructs a Newznab indexer. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Newznab {
	if client == nil {
		client = http.DefaultClient
	}
	return &Newznab{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (n *Newznab) Implementation() string { return "Newznab" }

// DefaultName satisfies providers.Provider.
func (n *Newznab) DefaultName() string { return "Newznab" }

// Settings satisfies providers.Provider.
func (n *Newznab) Settings() any { return &n.settings }

// Protocol satisfies indexer.Indexer.
func (n *Newznab) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }

// SupportsRss satisfies indexer.Indexer.
func (n *Newznab) SupportsRss() bool { return true }

// SupportsSearch satisfies indexer.Indexer.
func (n *Newznab) SupportsSearch() bool { return true }

// FetchRss retrieves the current TV RSS feed from the indexer.
// GET {BaseURL}{ApiPath}?t=tvsearch&cat={categories}&apikey={key}&dl=1
func (n *Newznab) FetchRss(ctx context.Context) ([]indexer.Release, error) {
	u := n.apiURL()
	q := u.Query()
	q.Set("t", "tvsearch")
	q.Set("apikey", n.settings.ApiKey)
	q.Set("dl", "1")
	if n.settings.Categories != "" {
		q.Set("cat", n.settings.Categories)
	}
	u.RawQuery = q.Encode()

	body, err := n.get(ctx, u.String())
	if err != nil {
		return nil, err
	}
	return parseRss(body)
}

// Search queries the indexer for releases matching req.
// GET {BaseURL}{ApiPath}?t=tvsearch&tvdbid={id}&season={s}&ep={e}&cat={categories}&apikey={key}
func (n *Newznab) Search(ctx context.Context, req indexer.SearchRequest) ([]indexer.Release, error) {
	u := n.apiURL()
	q := u.Query()
	q.Set("t", "tvsearch")
	q.Set("apikey", n.settings.ApiKey)
	if req.TvdbID != 0 {
		q.Set("tvdbid", strconv.FormatInt(req.TvdbID, 10))
	}
	if req.Season != 0 {
		q.Set("season", strconv.Itoa(req.Season))
	}
	if req.Episode != 0 {
		q.Set("ep", strconv.Itoa(req.Episode))
	}
	if n.settings.Categories != "" {
		q.Set("cat", n.settings.Categories)
	}
	u.RawQuery = q.Encode()

	body, err := n.get(ctx, u.String())
	if err != nil {
		return nil, err
	}
	return parseRss(body)
}

// Test verifies the indexer is reachable and the API key is accepted.
// GET {BaseURL}{ApiPath}?t=caps&apikey={key}
func (n *Newznab) Test(ctx context.Context) error {
	u := n.apiURL()
	q := u.Query()
	q.Set("t", "caps")
	q.Set("apikey", n.settings.ApiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("newznab: caps returned status %d", resp.StatusCode)
	}
	return nil
}

// apiURL builds the base URL for all API requests.
func (n *Newznab) apiURL() *url.URL {
	apiPath := n.settings.ApiPath
	if apiPath == "" {
		apiPath = "/api"
	}
	raw := n.settings.BaseURL + apiPath
	u, err := url.Parse(raw)
	if err != nil {
		// Fallback: return a minimal URL so callers can still attach query params.
		return &url.URL{Scheme: "http", Host: "invalid", Path: apiPath}
	}
	return u
}

// get executes an HTTP GET and returns the response body.
func (n *Newznab) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("newznab: request to %s returned status %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
