// Package torznab implements an indexer.Indexer for Torznab-compatible APIs.
// Torznab is Newznab for torrents: same XML format, torrent protocol, with
// additional <torznab:attr name="seeders"/> and <torznab:attr name="peers"/>
// attributes on each item.
package torznab

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Torznab is an indexer.Indexer that talks to Torznab-compatible APIs.
type Torznab struct {
	settings Settings
	client   *http.Client
}

// New constructs a Torznab indexer. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Torznab {
	if client == nil {
		client = http.DefaultClient
	}
	return &Torznab{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (t *Torznab) Implementation() string { return "Torznab" }

// DefaultName satisfies providers.Provider.
func (t *Torznab) DefaultName() string { return "Torznab" }

// Settings satisfies providers.Provider.
func (t *Torznab) Settings() any { return &t.settings }

// Protocol satisfies indexer.Indexer — torrents use the torrent protocol.
func (t *Torznab) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// SupportsRss satisfies indexer.Indexer.
func (t *Torznab) SupportsRss() bool { return true }

// SupportsSearch satisfies indexer.Indexer.
func (t *Torznab) SupportsSearch() bool { return true }

// FetchRss retrieves the current TV RSS feed from the Torznab indexer.
func (t *Torznab) FetchRss(ctx context.Context) ([]indexer.Release, error) {
	u := t.apiURL()
	q := u.Query()
	q.Set("t", "tvsearch")
	q.Set("apikey", t.settings.ApiKey)
	q.Set("dl", "1")
	if t.settings.Categories != "" {
		q.Set("cat", t.settings.Categories)
	}
	u.RawQuery = q.Encode()

	body, err := t.get(ctx, u.String())
	if err != nil {
		return nil, err
	}
	return t.parseAndFilter(body)
}

// Search queries the indexer for releases matching req.
func (t *Torznab) Search(ctx context.Context, req indexer.SearchRequest) ([]indexer.Release, error) {
	u := t.apiURL()
	q := u.Query()
	q.Set("t", "tvsearch")
	q.Set("apikey", t.settings.ApiKey)
	if req.TvdbID != 0 {
		q.Set("tvdbid", strconv.FormatInt(req.TvdbID, 10))
	}
	if req.Season != 0 {
		q.Set("season", strconv.Itoa(req.Season))
	}
	if req.Episode != 0 {
		q.Set("ep", strconv.Itoa(req.Episode))
	}
	if t.settings.Categories != "" {
		q.Set("cat", t.settings.Categories)
	}
	u.RawQuery = q.Encode()

	body, err := t.get(ctx, u.String())
	if err != nil {
		return nil, err
	}
	return t.parseAndFilter(body)
}

// Test verifies the indexer is reachable and the API key is accepted.
func (t *Torznab) Test(ctx context.Context) error {
	u := t.apiURL()
	q := u.Query()
	q.Set("t", "caps")
	q.Set("apikey", t.settings.ApiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torznab: caps returned status %d", resp.StatusCode)
	}
	return nil
}

// parseAndFilter parses a Torznab RSS response and applies the minSeeders filter.
func (t *Torznab) parseAndFilter(body []byte) ([]indexer.Release, error) {
	all, err := parseTorznabRss(body)
	if err != nil {
		return nil, err
	}
	if t.settings.MinSeeders <= 0 {
		return all, nil
	}
	filtered := all[:0]
	for _, r := range all {
		if r.Seeders >= t.settings.MinSeeders {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// apiURL builds the base URL for all API requests.
func (t *Torznab) apiURL() *url.URL {
	apiPath := t.settings.ApiPath
	if apiPath == "" {
		apiPath = "/api"
	}
	raw := t.settings.BaseURL + apiPath
	u, err := url.Parse(raw)
	if err != nil {
		return &url.URL{Scheme: "http", Host: "invalid", Path: apiPath}
	}
	return u
}

// get executes an HTTP GET and returns the response body.
func (t *Torznab) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torznab: request to %s returned status %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// --- XML parsing ---

// torznabAttr is a <torznab:attr name="..." value="..."/> or <newznab:attr .../> element.
type torznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// torznabEnclosure is the <enclosure> element.
type torznabEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
}

// torznabItem represents a single <item> in the RSS channel.
type torznabItem struct {
	Title     string           `xml:"title"`
	GUID      string           `xml:"guid"`
	Link      string           `xml:"link"`
	PubDate   string           `xml:"pubDate"`
	Enclosure torznabEnclosure `xml:"enclosure"`
	// Both the torznab and newznab namespaces use an "attr" local name.
	TorznabAttrs []torznabAttr `xml:"http://torznab.com/schemas/2015/feed attr"`
	NewznabAttrs []torznabAttr `xml:"http://www.newznab.com/DTD/2010/feeds/attributes/ attr"`
}

// allAttrs returns the combined attribute list from both namespaces.
func (i *torznabItem) allAttrs() []torznabAttr {
	return append(i.TorznabAttrs, i.NewznabAttrs...)
}

// torznabChannel is the <channel> element.
type torznabChannel struct {
	Items []torznabItem `xml:"item"`
}

// torznabRoot is the top-level <rss> element.
type torznabRoot struct {
	XMLName xml.Name       `xml:"rss"`
	Channel torznabChannel `xml:"channel"`
}

// parseTorznabRss parses a Torznab RSS XML response into a slice of Releases.
func parseTorznabRss(body []byte) ([]indexer.Release, error) {
	var root torznabRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, err
	}

	releases := make([]indexer.Release, 0, len(root.Channel.Items))
	for _, item := range root.Channel.Items {
		rel := indexer.Release{
			Title:    item.Title,
			GUID:     item.GUID,
			InfoURL:  item.Link,
			Protocol: indexer.ProtocolTorrent,
		}

		if item.Enclosure.URL != "" {
			rel.DownloadURL = item.Enclosure.URL
			if rel.Size == 0 && item.Enclosure.Length > 0 {
				rel.Size = item.Enclosure.Length
			}
		}

		if item.PubDate != "" {
			if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				rel.PublishDate = t
			} else if t, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				rel.PublishDate = t
			}
		}

		for _, attr := range item.allAttrs() {
			switch attr.Name {
			case "size":
				if n, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
					rel.Size = n
				}
			case "category":
				if n, err := strconv.Atoi(attr.Value); err == nil {
					rel.Categories = append(rel.Categories, n)
				}
			case "seeders":
				if n, err := strconv.Atoi(attr.Value); err == nil {
					rel.Seeders = n
				}
			case "peers":
				if n, err := strconv.Atoi(attr.Value); err == nil {
					rel.Leechers = n
				}
			}
		}

		releases = append(releases, rel)
	}
	return releases, nil
}
