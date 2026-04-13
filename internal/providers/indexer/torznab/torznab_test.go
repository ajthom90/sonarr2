package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// cannedTorznabRSS is a minimal Torznab RSS response with two items.
// It uses the torznab namespace for seeders/peers.
const cannedTorznabRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
  xmlns:torznab="http://torznab.com/schemas/2015/feed"
  xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
  <channel>
    <item>
      <title>Show.S01E01.720p.BluRay-GROUP</title>
      <guid>torrent-001</guid>
      <link>https://tracker.example.com/details/torrent-001</link>
      <pubDate>Sat, 12 Apr 2025 10:00:00 +0000</pubDate>
      <enclosure url="https://tracker.example.com/download/torrent-001.torrent" length="800000000" type="application/x-bittorrent"/>
      <torznab:attr name="seeders" value="42"/>
      <torznab:attr name="peers" value="10"/>
      <torznab:attr name="size" value="800000000"/>
      <torznab:attr name="category" value="5000"/>
    </item>
    <item>
      <title>Show.S01E02.1080p.WEB-GROUP</title>
      <guid>torrent-002</guid>
      <link>https://tracker.example.com/details/torrent-002</link>
      <pubDate>Sun, 13 Apr 2025 12:00:00 +0000</pubDate>
      <enclosure url="https://tracker.example.com/download/torrent-002.torrent" length="1200000000" type="application/x-bittorrent"/>
      <torznab:attr name="seeders" value="3"/>
      <torznab:attr name="peers" value="2"/>
      <torznab:attr name="size" value="1200000000"/>
      <torznab:attr name="category" value="5000"/>
    </item>
  </channel>
</rss>`

const cannedCaps = `<?xml version="1.0" encoding="UTF-8"?><caps><server version="1.0"/></caps>`

func newTestServer(handler http.Handler) (*http.Client, string) {
	srv := httptest.NewServer(handler)
	return srv.Client(), srv.URL
}

func TestTorznabImplementation(t *testing.T) {
	tz := New(Settings{BaseURL: "http://example.com", ApiKey: "k"}, nil)
	if tz.Implementation() != "Torznab" {
		t.Errorf("Implementation() = %q, want Torznab", tz.Implementation())
	}
	if tz.Protocol() != indexer.ProtocolTorrent {
		t.Errorf("Protocol() = %q, want torrent", tz.Protocol())
	}
	if !tz.SupportsRss() {
		t.Error("SupportsRss() should be true")
	}
	if !tz.SupportsSearch() {
		t.Error("SupportsSearch() should be true")
	}
}

func TestTorznabParseSeeders(t *testing.T) {
	releases, err := parseTorznabRss([]byte(cannedTorznabRSS))
	if err != nil {
		t.Fatalf("parseTorznabRss returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	r0 := releases[0]
	if r0.Title != "Show.S01E01.720p.BluRay-GROUP" {
		t.Errorf("title: got %q", r0.Title)
	}
	if r0.Seeders != 42 {
		t.Errorf("Seeders: got %d, want 42", r0.Seeders)
	}
	if r0.Leechers != 10 {
		t.Errorf("Leechers: got %d, want 10", r0.Leechers)
	}
	if r0.Size != 800000000 {
		t.Errorf("Size: got %d, want 800000000", r0.Size)
	}
	if r0.Protocol != indexer.ProtocolTorrent {
		t.Errorf("Protocol: got %q, want torrent", r0.Protocol)
	}

	r1 := releases[1]
	if r1.Seeders != 3 {
		t.Errorf("r1 Seeders: got %d, want 3", r1.Seeders)
	}
}

func TestTorznabFetchRss(t *testing.T) {
	client, baseURL := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedTorznabRSS))
	}))

	tz := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "testkey"}, client)
	releases, err := tz.FetchRss(context.Background())
	if err != nil {
		t.Fatalf("FetchRss returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
}

func TestTorznabMinSeedersFilter(t *testing.T) {
	client, baseURL := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedTorznabRSS))
	}))

	// MinSeeders=10 should keep item[0] (42 seeders) but drop item[1] (3 seeders).
	tz := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "testkey", MinSeeders: 10}, client)
	releases, err := tz.FetchRss(context.Background())
	if err != nil {
		t.Fatalf("FetchRss returned error: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release after minSeeders filter, got %d", len(releases))
	}
	if releases[0].Seeders != 42 {
		t.Errorf("expected the 42-seeder release, got %d seeders", releases[0].Seeders)
	}
}

func TestTorznabSearch(t *testing.T) {
	client, baseURL := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("t") != "tvsearch" {
			t.Errorf("expected t=tvsearch, got %s", q.Get("t"))
		}
		if q.Get("tvdbid") != "99999" {
			t.Errorf("expected tvdbid=99999, got %s", q.Get("tvdbid"))
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedTorznabRSS))
	}))

	tz := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "testkey"}, client)
	releases, err := tz.Search(context.Background(), indexer.SearchRequest{TvdbID: 99999, Season: 1, Episode: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
}

func TestTorznabTestSuccess(t *testing.T) {
	client, baseURL := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") != "caps" {
			t.Errorf("expected t=caps, got %s", r.URL.Query().Get("t"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedCaps))
	}))

	tz := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "testkey"}, client)
	if err := tz.Test(context.Background()); err != nil {
		t.Errorf("Test() returned unexpected error: %v", err)
	}
}

func TestTorznabTestFailure(t *testing.T) {
	client, baseURL := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))

	tz := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "badkey"}, client)
	if err := tz.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error for 403 response")
	}
}
