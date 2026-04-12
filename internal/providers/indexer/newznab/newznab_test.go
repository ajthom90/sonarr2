package newznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// cannedRSS is a minimal Newznab RSS response with two items.
const cannedRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
  <channel>
    <item>
      <title>Show.S01E01.720p.BluRay-GROUP</title>
      <guid>guid-001</guid>
      <link>https://indexer.example.com/details/guid-001</link>
      <pubDate>Sat, 12 Apr 2025 10:00:00 +0000</pubDate>
      <enclosure url="https://indexer.example.com/nzb/guid-001" length="1500000000" type="application/x-nzb"/>
      <newznab:attr name="size" value="1500000000"/>
      <newznab:attr name="category" value="5030"/>
      <newznab:attr name="tvdbid" value="12345"/>
    </item>
    <item>
      <title>Show.S01E02.1080p.WEB-GROUP</title>
      <guid>guid-002</guid>
      <link>https://indexer.example.com/details/guid-002</link>
      <pubDate>Sun, 13 Apr 2025 12:00:00 +0000</pubDate>
      <enclosure url="https://indexer.example.com/nzb/guid-002" length="2000000000" type="application/x-nzb"/>
      <newznab:attr name="size" value="2000000000"/>
      <newznab:attr name="category" value="5040"/>
    </item>
  </channel>
</rss>`

// cannedCaps is a minimal caps response.
const cannedCaps = `<?xml version="1.0" encoding="UTF-8"?>
<caps><server version="1.0"/></caps>`

func newTestClient(handler http.Handler) (*http.Client, string) {
	srv := httptest.NewServer(handler)
	return srv.Client(), srv.URL
}

func TestNewznabFetchRss(t *testing.T) {
	client, baseURL := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") != "tvsearch" {
			t.Errorf("expected t=tvsearch, got %s", r.URL.Query().Get("t"))
		}
		if r.URL.Query().Get("dl") != "1" {
			t.Errorf("expected dl=1, got %s", r.URL.Query().Get("dl"))
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedRSS))
	}))

	n := New(Settings{
		BaseURL:    baseURL,
		ApiPath:    "/api",
		ApiKey:     "testkey",
		Categories: "5030,5040",
	}, client)

	releases, err := n.FetchRss(context.Background())
	if err != nil {
		t.Fatalf("FetchRss returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	r0 := releases[0]
	if r0.Title != "Show.S01E01.720p.BluRay-GROUP" {
		t.Errorf("unexpected title: %q", r0.Title)
	}
	if r0.GUID != "guid-001" {
		t.Errorf("unexpected GUID: %q", r0.GUID)
	}
	if r0.DownloadURL != "https://indexer.example.com/nzb/guid-001" {
		t.Errorf("unexpected DownloadURL: %q", r0.DownloadURL)
	}
	if r0.Size != 1500000000 {
		t.Errorf("unexpected Size: %d", r0.Size)
	}
	if len(r0.Categories) != 1 || r0.Categories[0] != 5030 {
		t.Errorf("unexpected Categories: %v", r0.Categories)
	}
	wantDate := time.Date(2025, 4, 12, 10, 0, 0, 0, time.UTC)
	if !r0.PublishDate.Equal(wantDate) {
		t.Errorf("unexpected PublishDate: %v (want %v)", r0.PublishDate, wantDate)
	}

	r1 := releases[1]
	if r1.Title != "Show.S01E02.1080p.WEB-GROUP" {
		t.Errorf("unexpected title: %q", r1.Title)
	}
	if r1.Size != 2000000000 {
		t.Errorf("unexpected Size: %d", r1.Size)
	}
}

func TestNewznabSearch(t *testing.T) {
	client, baseURL := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("t") != "tvsearch" {
			t.Errorf("expected t=tvsearch, got %s", q.Get("t"))
		}
		if q.Get("tvdbid") != "12345" {
			t.Errorf("expected tvdbid=12345, got %s", q.Get("tvdbid"))
		}
		if q.Get("season") != "1" {
			t.Errorf("expected season=1, got %s", q.Get("season"))
		}
		if q.Get("ep") != "3" {
			t.Errorf("expected ep=3, got %s", q.Get("ep"))
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(cannedRSS))
	}))

	n := New(Settings{
		BaseURL: baseURL,
		ApiPath: "/api",
		ApiKey:  "testkey",
	}, client)

	releases, err := n.Search(context.Background(), indexer.SearchRequest{
		TvdbID:  12345,
		Season:  1,
		Episode: 3,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
}

func TestNewznabTestSuccess(t *testing.T) {
	client, baseURL := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") != "caps" {
			t.Errorf("expected t=caps, got %s", r.URL.Query().Get("t"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedCaps))
	}))

	n := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "testkey"}, client)
	if err := n.Test(context.Background()); err != nil {
		t.Errorf("Test() returned unexpected error: %v", err)
	}
}

func TestNewznabTestFailure(t *testing.T) {
	client, baseURL := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	n := New(Settings{BaseURL: baseURL, ApiPath: "/api", ApiKey: "badkey"}, client)
	err := n.Test(context.Background())
	if err == nil {
		t.Fatal("Test() should return an error for 401 response")
	}
}

func TestParseRss(t *testing.T) {
	releases, err := parseRss([]byte(cannedRSS))
	if err != nil {
		t.Fatalf("parseRss returned error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	r := releases[0]
	if r.Title != "Show.S01E01.720p.BluRay-GROUP" {
		t.Errorf("title: got %q", r.Title)
	}
	if r.GUID != "guid-001" {
		t.Errorf("GUID: got %q", r.GUID)
	}
	if r.DownloadURL != "https://indexer.example.com/nzb/guid-001" {
		t.Errorf("DownloadURL: got %q", r.DownloadURL)
	}
	if r.InfoURL != "https://indexer.example.com/details/guid-001" {
		t.Errorf("InfoURL: got %q", r.InfoURL)
	}
	if r.Size != 1500000000 {
		t.Errorf("Size: got %d", r.Size)
	}
	if len(r.Categories) != 1 || r.Categories[0] != 5030 {
		t.Errorf("Categories: got %v", r.Categories)
	}

	// Verify second item has category 5040.
	r2 := releases[1]
	if len(r2.Categories) != 1 || r2.Categories[0] != 5040 {
		t.Errorf("r2 Categories: got %v", r2.Categories)
	}
}

func TestParseRssEmptyChannel(t *testing.T) {
	const emptyRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
  <channel></channel>
</rss>`

	releases, err := parseRss([]byte(emptyRSS))
	if err != nil {
		t.Fatalf("parseRss returned error for empty channel: %v", err)
	}
	if len(releases) != 0 {
		t.Errorf("expected 0 releases, got %d", len(releases))
	}
}
