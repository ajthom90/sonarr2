package v6

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// setupTestPool returns an in-memory SQLite pool with migrations applied.
func setupTestPool(t *testing.T) (*db.SQLitePool, error) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, err
	}
	return pool, nil
}

// ---- History tests ----

func TestV6HistoryCursorPagination(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	store := history.NewSQLiteStore(pool)
	ctx := context.Background()

	// Seed 3 history entries.
	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, history.Entry{
			EpisodeID:   int64(i + 1),
			SeriesID:    1,
			SourceTitle: "Show S01E0" + itoa(int64(i+1)) + " 1080p",
			QualityName: "HDTV-1080p",
			EventType:   history.EventGrabbed,
			Date:        time.Now(),
		})
		if err != nil {
			t.Fatalf("Create history entry %d: %v", i, err)
		}
	}

	r := chi.NewRouter()
	h := newHistoryHandler(store, discardLog())
	mountHistory(r, h)

	req := httptest.NewRequest(http.MethodGet, "/history", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Must use v6 cursor pagination envelope.
	if _, ok := result["data"]; !ok {
		t.Error("missing 'data' key — history should use cursor pagination")
	}
	pagination, ok := result["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("missing 'pagination' key")
	}
	if pagination["limit"].(float64) != 50 {
		t.Errorf("limit = %v, want 50", pagination["limit"])
	}
	data := result["data"].([]any)
	if len(data) != 3 {
		t.Errorf("got %d history entries, want 3", len(data))
	}
}

func TestV6HistorySmallLimit(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	store := history.NewSQLiteStore(pool)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := store.Create(ctx, history.Entry{
			EpisodeID: int64(i + 1),
			SeriesID:  1,
			EventType: history.EventGrabbed,
			Date:      time.Now(),
		})
		if err != nil {
			t.Fatalf("Create history entry %d: %v", i, err)
		}
	}

	r := chi.NewRouter()
	h := newHistoryHandler(store, discardLog())
	mountHistory(r, h)

	req := httptest.NewRequest(http.MethodGet, "/history?limit=2", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	data := result["data"].([]any)
	if len(data) != 2 {
		t.Errorf("got %d history entries, want 2", len(data))
	}
	pagination := result["pagination"].(map[string]any)
	if pagination["hasMore"] != true {
		t.Errorf("hasMore = %v, want true", pagination["hasMore"])
	}
}

// ---- Command tests ----

func TestV6CommandPostReturnsCreated(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	queue := commands.NewSQLiteQueue(pool)

	r := chi.NewRouter()
	h := newCommandHandler(queue, discardLog())
	mountCommand(r, h)

	body := `{"name":"RescanSeries","body":{}}`
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["name"] != "RescanSeries" {
		t.Errorf("name = %v, want RescanSeries", result["name"])
	}
	if result["id"] == nil || result["id"].(float64) == 0 {
		t.Errorf("id should be non-zero, got %v", result["id"])
	}
	// Verify command shape fields.
	for _, field := range []string{"id", "name", "commandName", "status", "queued", "trigger", "priority"} {
		if _, ok := result[field]; !ok {
			t.Errorf("missing field %q in command response", field)
		}
	}
}

func TestV6CommandListReturnsArray(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	queue := commands.NewSQLiteQueue(pool)

	r := chi.NewRouter()
	h := newCommandHandler(queue, discardLog())
	mountCommand(r, h)

	req := httptest.NewRequest(http.MethodGet, "/command", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result []any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Empty list is fine.
	if result == nil {
		t.Error("expected non-nil array")
	}
}

func TestV6CommandGet404(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	queue := commands.NewSQLiteQueue(pool)

	r := chi.NewRouter()
	h := newCommandHandler(queue, discardLog())
	mountCommand(r, h)

	req := httptest.NewRequest(http.MethodGet, "/command/99999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

// ---- QualityProfile tests ----

func TestV6QualityProfileList(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	store := profiles.NewSQLiteQualityProfileStore(pool)
	defs := profiles.NewSQLiteQualityDefinitionStore(pool)

	r := chi.NewRouter()
	h := newQualityProfileHandler(store, defs, discardLog())
	mountQualityProfile(r, h)

	req := httptest.NewRequest(http.MethodGet, "/qualityprofile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Empty or seeded — just verify it's a list.
	if result == nil {
		t.Error("expected non-nil array")
	}
}

func TestV6QualityProfileGet404(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	store := profiles.NewSQLiteQualityProfileStore(pool)
	defs := profiles.NewSQLiteQualityDefinitionStore(pool)

	r := chi.NewRouter()
	h := newQualityProfileHandler(store, defs, discardLog())
	mountQualityProfile(r, h)

	req := httptest.NewRequest(http.MethodGet, "/qualityprofile/99999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

// ---- CustomFormat tests ----

func TestV6CustomFormatList(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	store := customformats.NewSQLiteStore(pool)

	r := chi.NewRouter()
	h := newCustomFormatHandler(store, discardLog())
	mountCustomFormat(r, h)

	req := httptest.NewRequest(http.MethodGet, "/customformat", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
}

// ---- Calendar tests ----

func TestV6CalendarReturnsEpisodesInRange(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 200, Title: "Calendar Show", Slug: "calendar-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/CalShow", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	inRange := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	outOfRange := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 1,
		Title: "In Range", AirDateUtc: &inRange, Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create in-range episode: %v", err)
	}
	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 2,
		Title: "Out of Range", AirDateUtc: &outOfRange, Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create out-of-range episode: %v", err)
	}

	r := chi.NewRouter()
	h := newCalendarHandler(lib.Episodes, discardLog())
	mountCalendar(r, h)

	req := httptest.NewRequest(http.MethodGet, "/calendar?start=2024-01-01&end=2024-04-30", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d episodes, want 1 (only in-range)", len(result))
	}
}

// ---- System status tests ----

func TestV6SystemStatusNoLegacyFields(t *testing.T) {
	r := chi.NewRouter()
	mountSystemStatus(r, nil)

	req := httptest.NewRequest(http.MethodGet, "/system/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// isNetCore must NOT be present in v6.
	if _, ok := result["isNetCore"]; ok {
		t.Error("isNetCore should NOT be present in v6 system/status response")
	}
	// But version and appName should be.
	for _, field := range []string{"appName", "version", "osName", "runtimeName"} {
		if _, ok := result[field]; !ok {
			t.Errorf("missing field %q in system/status response", field)
		}
	}
}

// ---- Parse tests ----

func TestV6ParseEndpoint(t *testing.T) {
	r := chi.NewRouter()
	mountParse(r)

	req := httptest.NewRequest(http.MethodGet, "/parse?title=Show.S01E01.1080p", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := result["parsedEpisodeInfo"]; !ok {
		t.Error("missing parsedEpisodeInfo")
	}
}

func TestV6ParseMissingTitle(t *testing.T) {
	r := chi.NewRouter()
	mountParse(r)

	req := httptest.NewRequest(http.MethodGet, "/parse", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}
