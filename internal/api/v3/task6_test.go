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

	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// --- Indexer tests ---

func TestIndexerCRUD(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	idxStore := indexer.NewSQLiteInstanceStore(pool)
	idxReg := indexer.NewRegistry()
	idxReg.Register("Newznab", func() indexer.Indexer {
		// stub — just need the registry for schema
		return &stubIndexer{}
	})

	r := chi.NewRouter()
	h := NewIndexerHandler(idxStore, idxReg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountIndexer(r, h)

	// POST create.
	body := `{"name":"My Indexer","implementation":"Newznab","fields":{},"enableRss":true,"enableAutomaticSearch":true,"enableInteractiveSearch":false,"priority":25}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/indexer", mustStringReader(body))
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
	if created["name"] != "My Indexer" {
		t.Errorf("name = %v, want My Indexer", created["name"])
	}
	id := int(created["id"].(float64))

	// GET by id.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/indexer/"+itoa(int64(id)), nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rr2.Code)
	}

	// GET list.
	req3 := httptest.NewRequest(http.MethodGet, "/api/v3/indexer", nil)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("GET list status = %d", rr3.Code)
	}
	var list []map[string]any
	if err := json.NewDecoder(rr3.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d indexers, want 1", len(list))
	}

	// GET schema.
	req4 := httptest.NewRequest(http.MethodGet, "/api/v3/indexer/schema", nil)
	rr4 := httptest.NewRecorder()
	r.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("GET schema status = %d", rr4.Code)
	}

	// DELETE.
	req5 := httptest.NewRequest(http.MethodDelete, "/api/v3/indexer/"+itoa(int64(id)), nil)
	rr5 := httptest.NewRecorder()
	r.ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", rr5.Code)
	}
}

// stubIndexer is a minimal indexer for tests that don't need real behavior.
type stubIndexer struct{}

func (s *stubIndexer) Implementation() string                                  { return "Newznab" }
func (s *stubIndexer) DefaultName() string                                     { return "Newznab" }
func (s *stubIndexer) Settings() any                                           { return nil }
func (s *stubIndexer) Test(ctx context.Context) error                          { return nil }
func (s *stubIndexer) Protocol() indexer.DownloadProtocol                      { return indexer.ProtocolUsenet }
func (s *stubIndexer) SupportsRss() bool                                       { return true }
func (s *stubIndexer) SupportsSearch() bool                                    { return true }
func (s *stubIndexer) FetchRss(ctx context.Context) ([]indexer.Release, error) { return nil, nil }
func (s *stubIndexer) Search(ctx context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, nil
}

// --- DownloadClient tests ---

func TestDownloadClientCRUD(t *testing.T) {
	pool, err := setupTestPool(t)
	if err != nil {
		t.Fatalf("setup pool: %v", err)
	}

	dcStore := downloadclient.NewSQLiteInstanceStore(pool)
	dcReg := downloadclient.NewRegistry()
	dcReg.Register("SABnzbd", func() downloadclient.DownloadClient {
		return &stubDCClient{}
	})

	r := chi.NewRouter()
	h := NewDownloadClientHandler(dcStore, dcReg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountDownloadClient(r, h)

	// POST create.
	body := `{"name":"SAB","implementation":"SABnzbd","fields":{},"enable":true,"priority":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/downloadclient", mustStringReader(body))
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
	id := int(created["id"].(float64))

	// GET schema.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/downloadclient/schema", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET schema status = %d", rr2.Code)
	}

	// DELETE.
	req3 := httptest.NewRequest(http.MethodDelete, "/api/v3/downloadclient/"+itoa(int64(id)), nil)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", rr3.Code)
	}
}

// stubDCClient is a minimal download client for tests.
type stubDCClient struct{}

func (s *stubDCClient) Implementation() string                                 { return "SABnzbd" }
func (s *stubDCClient) DefaultName() string                                    { return "SABnzbd" }
func (s *stubDCClient) Settings() any                                          { return nil }
func (s *stubDCClient) Test(ctx context.Context) error                         { return nil }
func (s *stubDCClient) Protocol() indexer.DownloadProtocol                     { return indexer.ProtocolUsenet }
func (s *stubDCClient) Add(_ context.Context, _, _ string) (string, error)     { return "dl-1", nil }
func (s *stubDCClient) Items(_ context.Context) ([]downloadclient.Item, error) { return nil, nil }
func (s *stubDCClient) Remove(_ context.Context, _ string, _ bool) error       { return nil }
func (s *stubDCClient) Status(_ context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{}, nil
}

// --- RootFolder tests ---

func TestRootFolderList(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	// Create series with shared root folder.
	type seriesSpec struct {
		tvdbID int64
		path   string
		slug   string
	}
	specs := []seriesSpec{
		{101, "/tv/Show A", "show-a"},
		{102, "/tv/Show B", "show-b"},
		{103, "/tv2/Show C", "show-c"},
	}
	for _, sp := range specs {
		_, err := lib.Series.Create(ctx, library.Series{
			TvdbID: sp.tvdbID, Title: sp.path, Slug: sp.slug,
			Status: "continuing", SeriesType: "standard",
			Path: sp.path, Monitored: true,
		})
		if err != nil {
			t.Fatalf("Create series %q: %v", sp.path, err)
		}
	}

	r := chi.NewRouter()
	h := NewRootFolderHandler(lib.Series, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountRootFolder(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/rootfolder", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should have 2 distinct root folders: /tv and /tv2.
	if len(result) != 2 {
		t.Errorf("got %d root folders, want 2", len(result))
	}
	for _, rf := range result {
		for _, field := range []string{"id", "path", "freeSpace", "unmappedFolders", "accessible"} {
			if _, ok := rf[field]; !ok {
				t.Errorf("missing field %q in rootfolder response", field)
			}
		}
	}
}

// --- Tag tests ---

func TestTagListReturnsEmpty(t *testing.T) {
	r := chi.NewRouter()
	MountTag(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/tag", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result []any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

// --- Health tests ---

func TestHealthReturnsEmpty(t *testing.T) {
	r := chi.NewRouter()
	checker := health.NewChecker() // empty checker returns empty results
	MountHealth(r, checker)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result []any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

// --- Parse tests ---

func TestParseReturnsFields(t *testing.T) {
	r := chi.NewRouter()
	MountParse(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/parse?title=The.Wire.S01E01.720p.BluRay", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["title"] != "The.Wire.S01E01.720p.BluRay" {
		t.Errorf("title = %v", result["title"])
	}
	if _, ok := result["parsedEpisodeInfo"]; !ok {
		t.Error("missing parsedEpisodeInfo field")
	}
	info := result["parsedEpisodeInfo"].(map[string]any)
	if info["seriesTitle"] == nil {
		t.Error("parsedEpisodeInfo.seriesTitle should be present")
	}
}

func TestParseMissingTitle(t *testing.T) {
	r := chi.NewRouter()
	MountParse(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/parse", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Wanted/Missing tests ---

func TestWantedMissingPagedEnvelope(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 30, Title: "Wanted Show", Slug: "wanted-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Wanted Show", Monitored: true,
	})

	// Create 3 monitored episodes without files.
	for i := 1; i <= 3; i++ {
		airDate := time.Date(2024, 1, i, 12, 0, 0, 0, time.UTC)
		_, err := lib.Episodes.Create(ctx, library.Episode{
			SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: int32(i),
			Title: "Episode", AirDateUtc: &airDate, Monitored: true,
		})
		if err != nil {
			t.Fatalf("create episode %d: %v", i, err)
		}
	}

	r := chi.NewRouter()
	h := NewWantedHandler(lib.Episodes, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountWanted(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/wanted/missing?page=1&pageSize=2", nil)
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
	if result["totalRecords"].(float64) != 3 {
		t.Errorf("totalRecords = %v, want 3", result["totalRecords"])
	}
	records := result["records"].([]any)
	if len(records) != 2 {
		t.Errorf("got %d records on page 1 (pageSize=2), want 2", len(records))
	}
}
