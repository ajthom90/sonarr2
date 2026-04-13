package v6

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
)

// itoa converts an int64 to a decimal string for use in test URLs.
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

// setupLibrary returns a *library.Library backed by an in-memory SQLite pool.
func setupLibrary(t *testing.T) *library.Library {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	bus := events.NewBus(4)
	lib, err := library.New(pool, bus)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return lib
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// ---- Series tests ----

func buildSeriesRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := newSeriesHandler(lib.Series, lib.Seasons, lib.Stats, discardLog())
	mountSeries(r, h)
	return r
}

func TestV6SeriesListCursorPagination(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	// Seed 3 series.
	for i := 1; i <= 3; i++ {
		_, err := lib.Series.Create(ctx, library.Series{
			TvdbID:     int64(i),
			Title:      "Show " + itoa(int64(i)),
			Slug:       "show-" + itoa(int64(i)),
			Status:     "continuing",
			SeriesType: "standard",
			Path:       "/tv/Show" + itoa(int64(i)),
			Monitored:  true,
		})
		if err != nil {
			t.Fatalf("Create series %d: %v", i, err)
		}
	}

	router := buildSeriesRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/series", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Must have "data" and "pagination" keys.
	if _, ok := result["data"]; !ok {
		t.Error("missing 'data' key in response")
	}
	pagination, ok := result["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("missing 'pagination' key or wrong type, got %T", result["pagination"])
	}
	if pagination["limit"].(float64) != 50 {
		t.Errorf("pagination.limit = %v, want 50", pagination["limit"])
	}
	if pagination["hasMore"] != false {
		t.Errorf("pagination.hasMore = %v, want false", pagination["hasMore"])
	}

	data := result["data"].([]any)
	if len(data) != 3 {
		t.Errorf("got %d series, want 3", len(data))
	}
}

func TestV6SeriesListSmallLimit(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, err := lib.Series.Create(ctx, library.Series{
			TvdbID:     int64(i),
			Title:      "Show " + itoa(int64(i)),
			Slug:       "show-paged-" + itoa(int64(i)),
			Status:     "continuing",
			SeriesType: "standard",
			Path:       "/tv/ShowP" + itoa(int64(i)),
			Monitored:  true,
		})
		if err != nil {
			t.Fatalf("Create series %d: %v", i, err)
		}
	}

	router := buildSeriesRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/series?limit=2", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	data := result["data"].([]any)
	if len(data) != 2 {
		t.Errorf("got %d series, want 2", len(data))
	}
	pagination := result["pagination"].(map[string]any)
	if pagination["hasMore"] != true {
		t.Errorf("hasMore = %v, want true", pagination["hasMore"])
	}
	if pagination["nextCursor"] == "" || pagination["nextCursor"] == nil {
		t.Error("nextCursor should be set when hasMore=true")
	}
}

func TestV6SeriesGet404(t *testing.T) {
	lib := setupLibrary(t)
	router := buildSeriesRouter(lib)

	req := httptest.NewRequest(http.MethodGet, "/series/99999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	var pd ProblemDetail
	if err := json.NewDecoder(rr.Body).Decode(&pd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pd.Status != 404 {
		t.Errorf("pd.Status = %d, want 404", pd.Status)
	}
	if pd.Type != "about:blank" {
		t.Errorf("pd.Type = %q, want about:blank", pd.Type)
	}
}

func TestV6SeriesCreate(t *testing.T) {
	lib := setupLibrary(t)
	router := buildSeriesRouter(lib)

	body := `{"title":"Succession","tvdbId":12345,"titleSlug":"succession","status":"ended","seriesType":"standard","path":"/tv/Succession","monitored":true}`
	req := httptest.NewRequest(http.MethodPost, "/series", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["title"] != "Succession" {
		t.Errorf("title = %v", result["title"])
	}
	// Verify no legacy field tvRageId.
	if _, ok := result["tvRageId"]; ok {
		t.Error("tvRageId should not be present in v6 response")
	}
	if _, ok := result["languageProfileId"]; ok {
		t.Error("languageProfileId should not be present in v6 response")
	}
}

// ---- Episode tests ----

func buildEpisodeRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := newEpisodeHandler(lib.Episodes, discardLog())
	mountEpisode(r, h)
	return r
}

func TestV6EpisodeListCursorPagination(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 1, Title: "Test Show", Slug: "test-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Test Show", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	airDate := time.Date(2024, 1, 15, 5, 0, 0, 0, time.UTC)
	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Pilot",
		AirDateUtc:    &airDate,
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode: %v", err)
	}

	router := buildEpisodeRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/episode?seriesId="+itoa(series.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := result["data"]; !ok {
		t.Error("missing 'data' key in response")
	}
	if _, ok := result["pagination"]; !ok {
		t.Error("missing 'pagination' key in response")
	}
	data := result["data"].([]any)
	if len(data) != 1 {
		t.Errorf("got %d episodes, want 1", len(data))
	}
}

func TestV6EpisodeGet404(t *testing.T) {
	lib := setupLibrary(t)
	router := buildEpisodeRouter(lib)

	req := httptest.NewRequest(http.MethodGet, "/episode/99999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

// ---- EpisodeFile tests ----

func buildEpisodeFileRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := newEpisodeFileHandler(lib.EpisodeFiles, lib.Series, discardLog())
	mountEpisodeFile(r, h)
	return r
}

func TestV6EpisodeFileListBySeries(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 10, Title: "File Show", Slug: "file-show-v6",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/File Show", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	_, err = lib.EpisodeFiles.Create(ctx, library.EpisodeFile{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		RelativePath: "Season 01/File.Show.S01E01.mkv",
		Size:         1500000000,
		DateAdded:    time.Now(),
		ReleaseGroup: "GROUP",
		QualityName:  "HDTV-1080p",
	})
	if err != nil {
		t.Fatalf("Create episode file: %v", err)
	}

	router := buildEpisodeFileRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/episodefile?seriesId="+itoa(series.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d files, want 1", len(result))
	}
}

func TestV6EpisodeFileGet404(t *testing.T) {
	lib := setupLibrary(t)
	router := buildEpisodeFileRouter(lib)

	req := httptest.NewRequest(http.MethodGet, "/episodefile/99999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}
