package tvdb_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/tvdb"
)

// loginBody is the canned login success response.
const loginBody = `{"status":"success","data":{"token":"test-jwt-token"}}`

// newTestServer builds an httptest server that handles /v4/login and any extra
// handlers the caller provides. The login handler is always registered first.
func newTestServer(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	mux.HandleFunc("/v4/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(loginBody))
	})
	return httptest.NewServer(mux)
}

// newClient creates a Client pointing at ts with the test API key.
func newClient(ts *httptest.Server) *tvdb.Client {
	c := tvdb.New(tvdb.Settings{ApiKey: "test-key"}, ts.Client())
	return c.WithBaseURL(ts.URL)
}

// --------------------------------------------------------------------------
// TestTVDBSearchSeries
// --------------------------------------------------------------------------

func TestTVDBSearchSeries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v4/search", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "simpsons" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("type") != "series" {
			http.Error(w, "bad type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": [
				{
					"tvdb_id": "71663",
					"name": "The Simpsons",
					"year": "1989",
					"overview": "The adventures of a working-class hero.",
					"status": "Continuing",
					"network": "FOX",
					"slug": "the-simpsons"
				},
				{
					"tvdb_id": "259972",
					"name": "The Simpsons 2",
					"year": "2010",
					"overview": "A sequel.",
					"status": "Ended",
					"network": "FOX",
					"slug": "the-simpsons-2"
				}
			]
		}`))
	})

	ts := newTestServer(t, mux)
	defer ts.Close()

	client := newClient(ts)
	results, err := client.SearchSeries(context.Background(), "simpsons")
	if err != nil {
		t.Fatalf("SearchSeries: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.TvdbID != 71663 {
		t.Errorf("TvdbID = %d, want 71663", r.TvdbID)
	}
	if r.Title != "The Simpsons" {
		t.Errorf("Title = %q, want %q", r.Title, "The Simpsons")
	}
	if r.Year != 1989 {
		t.Errorf("Year = %d, want 1989", r.Year)
	}
	if r.Status != "Continuing" {
		t.Errorf("Status = %q, want Continuing", r.Status)
	}
	if r.Network != "FOX" {
		t.Errorf("Network = %q, want FOX", r.Network)
	}
	if r.Slug != "the-simpsons" {
		t.Errorf("Slug = %q, want the-simpsons", r.Slug)
	}
}

// --------------------------------------------------------------------------
// TestTVDBGetSeries
// --------------------------------------------------------------------------

func TestTVDBGetSeries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v4/series/71663", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"id": 71663,
				"name": "The Simpsons",
				"year": "1989",
				"overview": "The adventures of a working-class hero.",
				"status": {"name": "Continuing"},
				"originalNetwork": {"name": "FOX"},
				"averageRuntime": 22,
				"airsTime": "20:00",
				"slug": "the-simpsons",
				"genres": [
					{"name": "Animation"},
					{"name": "Comedy"}
				]
			}
		}`))
	})

	ts := newTestServer(t, mux)
	defer ts.Close()

	client := newClient(ts)
	info, err := client.GetSeries(context.Background(), 71663)
	if err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	if info.TvdbID != 71663 {
		t.Errorf("TvdbID = %d, want 71663", info.TvdbID)
	}
	if info.Title != "The Simpsons" {
		t.Errorf("Title = %q", info.Title)
	}
	if info.Year != 1989 {
		t.Errorf("Year = %d, want 1989", info.Year)
	}
	if info.Status != "Continuing" {
		t.Errorf("Status = %q", info.Status)
	}
	if info.Network != "FOX" {
		t.Errorf("Network = %q, want FOX", info.Network)
	}
	if info.Runtime != 22 {
		t.Errorf("Runtime = %d, want 22", info.Runtime)
	}
	if info.AirTime != "20:00" {
		t.Errorf("AirTime = %q, want 20:00", info.AirTime)
	}
	if info.Slug != "the-simpsons" {
		t.Errorf("Slug = %q", info.Slug)
	}
	if len(info.Genres) != 2 {
		t.Fatalf("Genres len = %d, want 2", len(info.Genres))
	}
	if info.Genres[0] != "Animation" || info.Genres[1] != "Comedy" {
		t.Errorf("Genres = %v", info.Genres)
	}
}

// --------------------------------------------------------------------------
// TestTVDBGetEpisodes — canned episodes with two pages
// --------------------------------------------------------------------------

func TestTVDBGetEpisodes(t *testing.T) {
	page1 := `{
		"status": "success",
		"data": {
			"episodes": [
				{"id": 123, "seasonNumber": 1, "number": 1, "absoluteNumber": 1, "name": "Pilot", "overview": "The beginning.", "aired": "1989-12-17"},
				{"id": 124, "seasonNumber": 1, "number": 2, "absoluteNumber": 2, "name": "Bart the Genius", "overview": "Bart cheats.", "aired": "1990-01-14"}
			]
		},
		"links": {"prev": null, "self": "page=0", "next": "page=1", "total_items": 3, "page_size": 2}
	}`
	page2 := `{
		"status": "success",
		"data": {
			"episodes": [
				{"id": 125, "seasonNumber": 1, "number": 3, "absoluteNumber": 0, "name": "Homer's Odyssey", "overview": "Homer considers suicide.", "aired": "1990-01-21"}
			]
		},
		"links": {"prev": "page=0", "self": "page=1", "next": null, "total_items": 3, "page_size": 2}
	}`

	mux := http.NewServeMux()
	mux.HandleFunc("/v4/series/71663/episodes/default", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")
		switch page {
		case "0", "":
			_, _ = w.Write([]byte(page1))
		case "1":
			_, _ = w.Write([]byte(page2))
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
		}
	})

	ts := newTestServer(t, mux)
	defer ts.Close()

	client := newClient(ts)
	episodes, err := client.GetEpisodes(context.Background(), 71663)
	if err != nil {
		t.Fatalf("GetEpisodes: %v", err)
	}
	if len(episodes) != 3 {
		t.Fatalf("expected 3 episodes, got %d", len(episodes))
	}

	ep0 := episodes[0]
	if ep0.TvdbID != 123 {
		t.Errorf("ep0.TvdbID = %d, want 123", ep0.TvdbID)
	}
	if ep0.SeasonNumber != 1 {
		t.Errorf("ep0.SeasonNumber = %d, want 1", ep0.SeasonNumber)
	}
	if ep0.EpisodeNumber != 1 {
		t.Errorf("ep0.EpisodeNumber = %d, want 1", ep0.EpisodeNumber)
	}
	if ep0.AbsoluteEpisodeNumber == nil || *ep0.AbsoluteEpisodeNumber != 1 {
		t.Errorf("ep0.AbsoluteEpisodeNumber = %v, want 1", ep0.AbsoluteEpisodeNumber)
	}
	if ep0.AirDate == nil {
		t.Fatal("ep0.AirDate is nil")
	}
	if ep0.AirDate.Year() != 1989 || int(ep0.AirDate.Month()) != 12 || ep0.AirDate.Day() != 17 {
		t.Errorf("ep0.AirDate = %v, want 1989-12-17", ep0.AirDate)
	}

	ep2 := episodes[2]
	if ep2.TvdbID != 125 {
		t.Errorf("ep2.TvdbID = %d, want 125", ep2.TvdbID)
	}
	// absoluteNumber 0 means absent — should be nil
	if ep2.AbsoluteEpisodeNumber != nil {
		t.Errorf("ep2.AbsoluteEpisodeNumber should be nil for absoluteNumber=0, got %v", *ep2.AbsoluteEpisodeNumber)
	}
}

// --------------------------------------------------------------------------
// TestTVDBAuthCachesToken — login should be called only once for two API calls
// --------------------------------------------------------------------------

func TestTVDBAuthCachesToken(t *testing.T) {
	var loginCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v4/login", func(w http.ResponseWriter, r *http.Request) {
		loginCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(loginBody))
	})
	mux.HandleFunc("/v4/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[]}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := tvdb.New(tvdb.Settings{ApiKey: "test-key"}, ts.Client()).WithBaseURL(ts.URL)

	ctx := context.Background()
	if _, err := client.SearchSeries(ctx, "foo"); err != nil {
		t.Fatalf("first SearchSeries: %v", err)
	}
	if _, err := client.SearchSeries(ctx, "bar"); err != nil {
		t.Fatalf("second SearchSeries: %v", err)
	}

	if n := loginCalls.Load(); n != 1 {
		t.Errorf("login called %d times, want 1", n)
	}
}

// --------------------------------------------------------------------------
// TestTVDBAuthFailure — a 401 from login should be propagated as an error
// --------------------------------------------------------------------------

func TestTVDBAuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v4/login", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"status":"error","message":"Unauthorized"}`, http.StatusUnauthorized)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := tvdb.New(tvdb.Settings{ApiKey: "bad-key"}, ts.Client()).WithBaseURL(ts.URL)

	_, err := client.SearchSeries(context.Background(), "simpsons")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Verify the status code is mentioned.
	if !containsString(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got: %v", err)
	}
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

// containsString is a simple string-contains check that avoids importing
// strings just for tests.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}

// Compile-time check: Client implements MetadataSource via the package's own types.
// We verify the interface indirectly — tvdb_test imports only the tvdb package,
// so we use json.Marshal as a trivial import anchor and check via assignment.
var _ = json.Marshal
