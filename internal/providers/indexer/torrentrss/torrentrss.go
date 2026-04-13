// Package torrentrss implements an indexer.Indexer for generic torrent RSS feeds.
// It parses standard RSS <item> elements, using <enclosure> or <link> for the
// download URL. Search is not supported — use FetchRss for polling.
package torrentrss

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// TorrentRss is an indexer.Indexer that polls a generic torrent RSS feed.
type TorrentRss struct {
	settings Settings
	client   *http.Client
}

// New constructs a TorrentRss indexer. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *TorrentRss {
	if client == nil {
		client = http.DefaultClient
	}
	return &TorrentRss{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (t *TorrentRss) Implementation() string { return "TorrentRss" }

// DefaultName satisfies providers.Provider.
func (t *TorrentRss) DefaultName() string { return "TorrentRss" }

// Settings satisfies providers.Provider.
func (t *TorrentRss) Settings() any { return &t.settings }

// Protocol satisfies indexer.Indexer.
func (t *TorrentRss) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// SupportsRss satisfies indexer.Indexer.
func (t *TorrentRss) SupportsRss() bool { return true }

// SupportsSearch satisfies indexer.Indexer — generic RSS feeds have no search API.
func (t *TorrentRss) SupportsSearch() bool { return false }

// FetchRss fetches and parses the configured RSS feed.
func (t *TorrentRss) FetchRss(ctx context.Context) ([]indexer.Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.settings.FeedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("torrentrss: failed to build request: %w", err)
	}
	if t.settings.Cookie != "" {
		req.Header.Set("Cookie", t.settings.Cookie)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("torrentrss: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torrentrss: feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("torrentrss: failed to read response: %w", err)
	}
	return parseRss(body)
}

// Search is not supported for generic RSS feeds.
func (t *TorrentRss) Search(_ context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("torrentrss: search is not supported for generic RSS feeds")
}

// Test verifies the feed URL is reachable and returns a 200 response.
func (t *TorrentRss) Test(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.settings.FeedURL, nil)
	if err != nil {
		return fmt.Errorf("torrentrss: failed to build request: %w", err)
	}
	if t.settings.Cookie != "" {
		req.Header.Set("Cookie", t.settings.Cookie)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("torrentrss: connection failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torrentrss: feed returned status %d", resp.StatusCode)
	}
	return nil
}

// --- XML parsing ---

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
}

type rssItem struct {
	Title     string       `xml:"title"`
	GUID      string       `xml:"guid"`
	Link      string       `xml:"link"`
	PubDate   string       `xml:"pubDate"`
	Enclosure rssEnclosure `xml:"enclosure"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

// parseRss parses a standard RSS feed body into a slice of Releases.
func parseRss(body []byte) ([]indexer.Release, error) {
	var root rssRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, err
	}

	releases := make([]indexer.Release, 0, len(root.Channel.Items))
	for _, item := range root.Channel.Items {
		rel := indexer.Release{
			Title:    item.Title,
			GUID:     item.GUID,
			Protocol: indexer.ProtocolTorrent,
		}

		// Prefer enclosure URL; fall back to <link>.
		if item.Enclosure.URL != "" {
			rel.DownloadURL = item.Enclosure.URL
			rel.Size = item.Enclosure.Length
		} else {
			rel.DownloadURL = item.Link
			rel.InfoURL = item.Link
		}

		if item.PubDate != "" {
			if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				rel.PublishDate = t
			} else if t, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				rel.PublishDate = t
			}
		}

		releases = append(releases, rel)
	}
	return releases, nil
}
