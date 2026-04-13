package v3

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// --- QualityProfile tests ---

func TestQualityProfileListReturnsJSON(t *testing.T) {
	lib := setupLibrary(t)

	// setupLibrary seeds DB but no quality profiles are seeded in the test DB.
	// QualityProfileStore is obtained from a fresh library helper here.
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	store := setupQualityProfileStore(t, pool)

	r := chi.NewRouter()
	h := NewQualityProfileHandler(store.profiles, store.defs, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountQualityProfile(r, h)
	_ = lib // library seeded but profiles come from store

	req := httptest.NewRequest(http.MethodGet, "/api/v3/qualityprofile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Fields present (even if empty list).
	t.Logf("quality profiles: %d", len(result))
}

// --- QualityDefinition tests ---

func TestQualityDefinitionListReturnsJSON(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	store := setupQualityProfileStore(t, pool)

	r := chi.NewRouter()
	h := NewQualityDefinitionHandler(store.defs, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountQualityDefinition(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/qualitydefinition", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Quality definitions are seeded by migrations; expect >= 0.
	t.Logf("quality definitions: %d", len(result))
	if len(result) > 0 {
		for _, field := range []string{"id", "quality", "title", "minSize", "maxSize"} {
			if _, ok := result[0][field]; !ok {
				t.Errorf("missing field %q in quality definition", field)
			}
		}
	}
}

// --- CustomFormat tests ---

func TestCustomFormatListReturnsJSON(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	cfStore := setupCFStore(t, pool)

	r := chi.NewRouter()
	h := NewCustomFormatHandler(cfStore, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountCustomFormat(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/customformat", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	t.Logf("custom formats: %d", len(result))
}

// --- Command tests ---

func TestCommandEnqueueAndGet(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	cmdQueue := setupCommandQueue(t, pool)

	r := chi.NewRouter()
	h := NewCommandHandler(cmdQueue, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountCommand(r, h)

	// POST to enqueue.
	body := `{"name":"RssSync","body":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/command",
		mustStringReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}

	var created map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created["name"] != "RssSync" {
		t.Errorf("name = %v, want RssSync", created["name"])
	}
	for _, field := range []string{"id", "name", "commandName", "status", "queued", "trigger", "priority"} {
		if _, ok := created[field]; !ok {
			t.Errorf("missing field %q in command response", field)
		}
	}

	id := int64(created["id"].(float64))

	// GET the command by id.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/command/"+itoa(id), nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rr2.Code)
	}

	// GET list.
	req3 := httptest.NewRequest(http.MethodGet, "/api/v3/command", nil)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want 200", rr3.Code)
	}
	var list []map[string]any
	if err := json.NewDecoder(rr3.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one command in list")
	}
}

// --- History tests ---

func TestHistoryListPagedEnvelope(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}
	histStore := setupHistoryStore(t, pool)

	r := chi.NewRouter()
	h := NewHistoryHandler(histStore, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountHistory(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/history?page=1&pageSize=10", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, field := range []string{"page", "pageSize", "sortKey", "sortDirection", "totalRecords", "records"} {
		if _, ok := result[field]; !ok {
			t.Errorf("missing paged envelope field %q", field)
		}
	}
	if result["page"].(float64) != 1 {
		t.Errorf("page = %v, want 1", result["page"])
	}
	if result["pageSize"].(float64) != 10 {
		t.Errorf("pageSize = %v, want 10", result["pageSize"])
	}
}

// --- Calendar tests ---

func TestCalendarFilterByDateRange(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 20, Title: "Calendar Show", Slug: "calendar-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Calendar Show", Monitored: true,
	})

	// Episode in range.
	inRange := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	_, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 1,
		Title: "In Range", AirDateUtc: &inRange, Monitored: true,
	})
	if err != nil {
		t.Fatalf("create in-range episode: %v", err)
	}

	// Episode out of range.
	outRange := time.Date(2024, 2, 10, 12, 0, 0, 0, time.UTC)
	_, err = lib.Episodes.Create(ctx, library.Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 2,
		Title: "Out Range", AirDateUtc: &outRange, Monitored: true,
	})
	if err != nil {
		t.Fatalf("create out-range episode: %v", err)
	}

	r := chi.NewRouter()
	h := NewCalendarHandler(lib.Episodes, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountCalendar(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/calendar?start=2024-01-01&end=2024-01-31", nil)
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
		t.Errorf("got %d calendar episodes, want 1", len(result))
	}
	if len(result) > 0 && result[0]["title"] != "In Range" {
		t.Errorf("title = %v, want 'In Range'", result[0]["title"])
	}
}
