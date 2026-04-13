package torrentrss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

const cannedFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Show.S01E01.720p.BluRay-GROUP</title>
      <guid>https://tracker.example.com/torrent/1</guid>
      <link>https://tracker.example.com/torrent/1</link>
      <pubDate>Sat, 12 Apr 2025 10:00:00 +0000</pubDate>
      <enclosure url="https://tracker.example.com/download/1.torrent" length="900000000" type="application/x-bittorrent"/>
    </item>
    <item>
      <title>Show.S01E02.1080p.WEB-GROUP</title>
      <guid>https://tracker.example.com/torrent/2</guid>
      <link>https://tracker.example.com/download/2.torrent</link>
      <pubDate>Sun, 13 Apr 2025 12:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func TestTorrentRssImplementation(t *testing.T) {
	tr := New(Settings{FeedURL: "http://example.com/rss"}, nil)
	if tr.Implementation() != "TorrentRss" {
		t.Errorf("Implementation() = %q, want TorrentRss", tr.Implementation())
	}
	if tr.Protocol() != indexer.ProtocolTorrent {
		t.Errorf("Protocol() = %q, want torrent", tr.Protocol())
	}
	if !tr.SupportsRss() {
		t.Error("SupportsRss() should be true")
	}
	if tr.SupportsSearch() {
		t.Error("SupportsSearch() should be false")
	}
}

func TestTorrentRssFetchRss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedFeed))
	}))
	defer srv.Close()

	tr := New(Settings{FeedURL: srv.URL + "/rss"}, srv.Client())
	releases, err := tr.FetchRss(context.Background())
	if err != nil {
		t.Fatalf("FetchRss returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	r0 := releases[0]
	if r0.Title != "Show.S01E01.720p.BluRay-GROUP" {
		t.Errorf("title: got %q", r0.Title)
	}
	if r0.DownloadURL != "https://tracker.example.com/download/1.torrent" {
		t.Errorf("DownloadURL: got %q", r0.DownloadURL)
	}
	if r0.Size != 900000000 {
		t.Errorf("Size: got %d, want 900000000", r0.Size)
	}
	if r0.Protocol != indexer.ProtocolTorrent {
		t.Errorf("Protocol: got %q", r0.Protocol)
	}

	// Second item has no enclosure — falls back to link.
	r1 := releases[1]
	if r1.DownloadURL != "https://tracker.example.com/download/2.torrent" {
		t.Errorf("r1 DownloadURL: got %q", r1.DownloadURL)
	}
}

func TestTorrentRssCookieHeader(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedFeed))
	}))
	defer srv.Close()

	tr := New(Settings{FeedURL: srv.URL + "/rss", Cookie: "uid=42; pass=secret"}, srv.Client())
	_, err := tr.FetchRss(context.Background())
	if err != nil {
		t.Fatalf("FetchRss returned error: %v", err)
	}
	if gotCookie != "uid=42; pass=secret" {
		t.Errorf("Cookie header: got %q, want uid=42; pass=secret", gotCookie)
	}
}

func TestTorrentRssSearchUnsupported(t *testing.T) {
	tr := New(Settings{FeedURL: "http://example.com/rss"}, nil)
	_, err := tr.Search(context.Background(), indexer.SearchRequest{})
	if err == nil {
		t.Fatal("Search() should return an error for TorrentRss")
	}
}

func TestTorrentRssTestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedFeed))
	}))
	defer srv.Close()

	tr := New(Settings{FeedURL: srv.URL + "/rss"}, srv.Client())
	if err := tr.Test(context.Background()); err != nil {
		t.Errorf("Test() returned unexpected error: %v", err)
	}
}

func TestTorrentRssTestFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	tr := New(Settings{FeedURL: srv.URL + "/rss"}, srv.Client())
	if err := tr.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error for 403 response")
	}
}

func TestParseRssEmptyChannel(t *testing.T) {
	const empty = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel></channel></rss>`
	releases, err := parseRss([]byte(empty))
	if err != nil {
		t.Fatalf("parseRss returned error: %v", err)
	}
	if len(releases) != 0 {
		t.Errorf("expected 0 releases, got %d", len(releases))
	}
}
