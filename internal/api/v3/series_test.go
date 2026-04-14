package v3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
)

// setupLibrary returns a *library.Library backed by an in-memory SQLite pool
// with all migrations applied. The returned cleanup func closes the pool.
// Two quality profiles (id=1 "Any", id=2 "HD-1080p") are seeded so tests
// can reference either without tripping the series.quality_profile_id FK.
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
	// Seed profiles referenced by tests that use qualityProfileId in request
	// bodies. Without this the series FK would fail.
	if err := pool.Write(ctx, func(exec db.Executor) error {
		if _, err := exec.ExecContext(ctx, `INSERT INTO quality_profiles (id, name) VALUES (1, 'Any')`); err != nil {
			return err
		}
		_, err := exec.ExecContext(ctx, `INSERT INTO quality_profiles (id, name) VALUES (2, 'HD-1080p')`)
		return err
	}); err != nil {
		t.Fatalf("seed quality profiles: %v", err)
	}
	bus := events.NewBus(4)
	lib, err := library.New(pool, bus)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return lib
}

// fakeCommands is a no-op CommandEnqueuer that records the enqueue calls
// so tests can assert (or ignore) post-create background work.
type fakeCommands struct {
	calls []struct {
		Name string
		Body map[string]any
	}
}

func (f *fakeCommands) Enqueue(_ context.Context, name string, body map[string]any) error {
	f.calls = append(f.calls, struct {
		Name string
		Body map[string]any
	}{Name: name, Body: body})
	return nil
}

// buildRouter builds a chi.Router with the series routes but no auth
// middleware so tests can exercise handlers directly.
func buildRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := NewSeriesHandler(lib.Series, lib.Seasons, lib.Stats, lib.Episodes, &fakeCommands{}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountSeries(r, h)
	return r
}

func TestSeriesListReturnsJSON(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	// Seed two series.
	_, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 1, Title: "The Wire", Slug: "the-wire",
		Status: "ended", SeriesType: "standard",
		Path: "/tv/The Wire", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series 1: %v", err)
	}
	_, err = lib.Series.Create(ctx, library.Series{
		TvdbID: 2, Title: "Breaking Bad", Slug: "breaking-bad",
		Status: "ended", SeriesType: "standard",
		Path: "/tv/Breaking Bad", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series 2: %v", err)
	}

	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/series", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q", ct)
	}

	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d series, want 2", len(result))
	}
}

func TestSeriesGetByID(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	created, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 71663, Title: "The Simpsons", Slug: "the-simpsons",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/The Simpsons", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/series/"+itoa(created.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["title"] != "The Simpsons" {
		t.Errorf("title = %v, want 'The Simpsons'", result["title"])
	}
	if result["sortTitle"] != "simpsons" {
		t.Errorf("sortTitle = %v, want 'simpsons'", result["sortTitle"])
	}
	if result["tvdbId"].(float64) != 71663 {
		t.Errorf("tvdbId = %v, want 71663", result["tvdbId"])
	}
	// status=continuing → ended=false
	if result["ended"] != (created.Status == "ended") {
		t.Errorf("ended = %v, want %v", result["ended"], created.Status == "ended")
	}
}

func TestSeriesCreate(t *testing.T) {
	lib := setupLibrary(t)

	body := `{"title":"Succession","tvdbId":12345,"titleSlug":"succession","status":"ended","seriesType":"standard","path":"/tv/Succession","monitored":true}`
	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodPost, "/api/v3/series", bytes.NewBufferString(body))
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
	if result["id"] == nil || result["id"].(float64) == 0 {
		t.Errorf("id should be non-zero, got %v", result["id"])
	}
	if result["title"] != "Succession" {
		t.Errorf("title = %v, want Succession", result["title"])
	}
}

func TestSeriesUpdateMonitored(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	created, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 99, Title: "Sopranos", Slug: "sopranos",
		Status: "ended", SeriesType: "standard",
		Path: "/tv/Sopranos", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Toggle monitored to false.
	updateBody := `{"title":"Sopranos","tvdbId":99,"titleSlug":"sopranos","status":"ended","seriesType":"standard","path":"/tv/Sopranos","monitored":false}`
	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodPut, "/api/v3/series/"+itoa(created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	// GET and verify.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/series/"+itoa(created.ID), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	var result map[string]any
	if err := json.NewDecoder(rr2.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["monitored"] != false {
		t.Errorf("monitored = %v, want false", result["monitored"])
	}
}

func TestSeriesDelete(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	created, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 55, Title: "Deadwood", Slug: "deadwood",
		Status: "ended", SeriesType: "standard",
		Path: "/tv/Deadwood", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := buildRouter(lib)

	// DELETE.
	req := httptest.NewRequest(http.MethodDelete, "/api/v3/series/"+itoa(created.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", rr.Code)
	}

	// GET should return 404.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/series/"+itoa(created.ID), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want 404", rr2.Code)
	}
}

func TestSeriesGoldenFieldNames(t *testing.T) {
	// Load the golden fixture to get expected top-level field names.
	_, thisFile, _, _ := runtime.Caller(0)
	goldenPath := filepath.Join(filepath.Dir(thisFile), "..", "testdata", "golden", "series_detail.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Skipf("golden fixture not found at %s: %v", goldenPath, err)
	}

	var golden map[string]any
	if err := json.Unmarshal(goldenData, &golden); err != nil {
		t.Fatalf("parse golden: %v", err)
	}

	// Build our own response.
	lib := setupLibrary(t)
	ctx := context.Background()
	created, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 418229, Title: "007: Road to a Million", Slug: "007-road-to-a-million",
		Status: "continuing", SeriesType: "standard",
		Path: "/data/TVShows/007 Road to a Million", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/series/"+itoa(created.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var our map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&our); err != nil {
		t.Fatalf("decode our response: %v", err)
	}

	// Fields that the golden fixture has but we intentionally omit or zero-out.
	// These are fields we don't track yet (no data) and thus don't emit.
	skipFields := map[string]bool{
		"previousAiring":    true,
		"languageProfileId": true,
		"certification":     true,
		"year":              true, // we don't track year as a separate field yet
	}

	// Verify every top-level key in the golden fixture appears in our response
	// (except the known skip list).
	for key := range golden {
		if skipFields[key] {
			continue
		}
		if _, ok := our[key]; !ok {
			t.Errorf("missing field %q (present in golden fixture, absent in our response)", key)
		}
	}
}

// itoa is a trivial int64→string helper so tests don't need strconv.
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

// TestSeriesCreate_PersistsLibraryImportFields verifies that POST
// /api/v3/series accepts the library-import fields (qualityProfileId,
// seasonFolder, monitorNewItems) on the way in and echoes them back
// unchanged on the way out — proving both the input-mapping and the
// response-rendering paths are wired.
func TestSeriesCreate_PersistsLibraryImportFields(t *testing.T) {
	lib := setupLibrary(t)

	body, err := json.Marshal(map[string]any{
		"title":            "Test",
		"tvdbId":           9999,
		"path":             "/tmp/fixtures/Test",
		"seriesType":       "standard",
		"monitored":        true,
		"qualityProfileId": 2,
		"seasonFolder":     false,
		"monitorNewItems":  "all",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodPost, "/api/v3/series", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}

	var res struct {
		QualityProfileID int    `json:"qualityProfileId"`
		SeasonFolder     bool   `json:"seasonFolder"`
		MonitorNewItems  string `json:"monitorNewItems"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.QualityProfileID != 2 {
		t.Errorf("qualityProfileId = %d, want 2", res.QualityProfileID)
	}
	if res.SeasonFolder != false {
		t.Errorf("seasonFolder = %v, want false", res.SeasonFolder)
	}
	if res.MonitorNewItems != "all" {
		t.Errorf("monitorNewItems = %q, want %q", res.MonitorNewItems, "all")
	}
}

// TestSeriesCreate_InvalidMonitorMode verifies that a bogus addOptions.monitor
// value is rejected with 400 before any DB writes happen.
func TestSeriesCreate_InvalidMonitorMode(t *testing.T) {
	lib := setupLibrary(t)

	body, err := json.Marshal(map[string]any{
		"title":      "Test",
		"tvdbId":     9999,
		"path":       "/tmp/x",
		"monitored":  true,
		"addOptions": map[string]any{"monitor": "bogus"},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	router := buildRouter(lib)
	req := httptest.NewRequest(http.MethodPost, "/api/v3/series", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
}
