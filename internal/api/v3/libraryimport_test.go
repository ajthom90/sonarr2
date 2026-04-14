package v3_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	v3 "github.com/ajthom90/sonarr2/internal/api/v3"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// mockMetadataSource is a test double for metadatasource.MetadataSource
// that counts invocations, tracks peak concurrency, and returns results
// from a term→results map. Unknown terms return an empty result set
// (not an error) so handlers treat them like "no TVDB match."
type mockMetadataSource struct {
	// calls is the total SearchSeries invocation count.
	calls int64
	// delay is the synthetic sleep per call, used to make concurrency
	// measurable. Set to 0 for tests that don't care about timing.
	delay time.Duration

	mu       sync.Mutex
	inFlight int
	peak     int
	results  map[string][]metadatasource.SeriesSearchResult
	// errorsFor is a set of terms that should return a non-nil error
	// instead of a result slice — used to exercise soft-failure paths.
	errorsFor map[string]bool
}

func newMockMetadataSource() *mockMetadataSource {
	return &mockMetadataSource{
		results:   make(map[string][]metadatasource.SeriesSearchResult),
		errorsFor: make(map[string]bool),
	}
}

func (m *mockMetadataSource) SearchSeries(_ context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	atomic.AddInt64(&m.calls, 1)
	m.mu.Lock()
	m.inFlight++
	if m.inFlight > m.peak {
		m.peak = m.inFlight
	}
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.inFlight--
		m.mu.Unlock()
	}()
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.errorsFor[query] {
		return nil, fmt.Errorf("synthetic TVDB error for %q", query)
	}
	return m.results[query], nil
}

func (m *mockMetadataSource) GetSeries(_ context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	return metadatasource.SeriesInfo{TvdbID: tvdbID}, nil
}

func (m *mockMetadataSource) GetEpisodes(_ context.Context, _ int64) ([]metadatasource.EpisodeInfo, error) {
	return nil, nil
}

// libraryImportRouter mounts just the libraryimport endpoint so tests can
// exercise GET /api/v3/libraryimport/scan without wiring the whole app.
func (h *testHarness) libraryImportRouter(source metadatasource.MetadataSource) chi.Router {
	r := chi.NewRouter()
	v3.MountLibraryImport(r, h.rootFolder, h.series, h.hostConfig, source)
	return r
}

// seedTvdbKey writes a TvdbApiKey into host_config so non-preview scans
// don't short-circuit with a 503.
func (h *testHarness) seedTvdbKey(t *testing.T) {
	t.Helper()
	if err := h.hostConfig.Upsert(context.Background(), hostconfig.HostConfig{
		APIKey:         "test-api-key",
		AuthMode:       "forms",
		MigrationState: "migrated",
		TvdbApiKey:     "test-tvdb-key",
	}); err != nil {
		t.Fatalf("seed host_config: %v", err)
	}
}

// TestLibraryImportScan_HappyPath covers the common case: 2 matched
// folders, 1 unparseable folder (no TVDB result), 1 dotfolder (skipped),
// and 1 file (skipped). The response should contain exactly 3 entries:
// the two matches plus the unparseable one (with tvdbMatch=null).
func TestLibraryImportScan_HappyPath(t *testing.T) {
	h := newTestHarness(t)
	h.seedTvdbKey(t)

	dir := t.TempDir()
	for _, sub := range []string{
		"Breaking Bad (2008)",
		"The Wire (2002)",
		"Gibberish Folder Name",
		".hiddenfolder",
	} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	rf, err := h.rootFolder.Create(context.Background(), dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	source := newMockMetadataSource()
	source.results["Breaking Bad"] = []metadatasource.SeriesSearchResult{
		{TvdbID: 81189, Title: "Breaking Bad", Year: 2008, Overview: "meth"},
	}
	source.results["The Wire"] = []metadatasource.SeriesSearchResult{
		{TvdbID: 79126, Title: "The Wire", Year: 2002, Overview: "baltimore"},
	}
	// Gibberish Folder Name deliberately has no result → tvdbMatch=null.

	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId="+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var got []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3; body = %s", len(got), rr.Body.String())
	}

	byName := make(map[string]map[string]any, len(got))
	for _, e := range got {
		byName[e["folderName"].(string)] = e
	}
	if _, ok := byName[".hiddenfolder"]; ok {
		t.Errorf("dotfolder was not skipped: %+v", byName[".hiddenfolder"])
	}
	if _, ok := byName["notes.txt"]; ok {
		t.Errorf("file was not skipped: %+v", byName["notes.txt"])
	}

	bb, ok := byName["Breaking Bad (2008)"]
	if !ok {
		t.Fatalf("missing Breaking Bad entry; got keys = %v", keys(byName))
	}
	match, ok := bb["tvdbMatch"].(map[string]any)
	if !ok {
		t.Fatalf("Breaking Bad tvdbMatch = %v, want object", bb["tvdbMatch"])
	}
	if id, _ := match["tvdbId"].(float64); int64(id) != 81189 {
		t.Errorf("Breaking Bad tvdbId = %v, want 81189", match["tvdbId"])
	}
	if bb["relativePath"] != "Breaking Bad (2008)" {
		t.Errorf("Breaking Bad relativePath = %v, want %q", bb["relativePath"], "Breaking Bad (2008)")
	}
	wantAbs := filepath.Join(dir, "Breaking Bad (2008)")
	if bb["absolutePath"] != wantAbs {
		t.Errorf("Breaking Bad absolutePath = %v, want %q", bb["absolutePath"], wantAbs)
	}
	if imported, _ := bb["alreadyImported"].(bool); imported {
		t.Errorf("Breaking Bad alreadyImported = true, want false")
	}

	gib, ok := byName["Gibberish Folder Name"]
	if !ok {
		t.Fatalf("missing Gibberish entry")
	}
	if gib["tvdbMatch"] != nil {
		t.Errorf("Gibberish tvdbMatch = %v, want nil", gib["tvdbMatch"])
	}
}

// TestLibraryImportScan_TVDBKeyMissing verifies that a non-preview scan
// with no TvdbApiKey returns 503 and surfaces fixPath so the frontend
// can deep-link to Settings → General.
func TestLibraryImportScan_TVDBKeyMissing(t *testing.T) {
	h := newTestHarness(t)
	// Intentionally skip seedTvdbKey.

	dir := t.TempDir()
	rf, err := h.rootFolder.Create(context.Background(), dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	source := newMockMetadataSource()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId="+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["fixPath"] != "/settings/general" {
		t.Errorf("fixPath = %v, want %q", body["fixPath"], "/settings/general")
	}
	if got, _ := body["message"].(string); got == "" {
		t.Errorf("message is empty")
	}
	if got := atomic.LoadInt64(&source.calls); got != 0 {
		t.Errorf("source.calls = %d, want 0 (short-circuit before TVDB)", got)
	}
}

// TestLibraryImportScan_PreviewSkipsTVDB verifies previewOnly=true returns
// 200 with all tvdbMatch=null and never touches the metadata source —
// even when no TvdbApiKey is configured.
func TestLibraryImportScan_PreviewSkipsTVDB(t *testing.T) {
	h := newTestHarness(t)
	// No TVDB key — preview must still work.

	dir := t.TempDir()
	for _, sub := range []string{"Show A", "Show B"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	rf, err := h.rootFolder.Create(context.Background(), dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	source := newMockMetadataSource()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId="+strconv.FormatInt(rf.ID, 10)+"&previewOnly=true", nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var got []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	for _, e := range got {
		if e["tvdbMatch"] != nil {
			t.Errorf("%s tvdbMatch = %v, want nil (preview)", e["folderName"], e["tvdbMatch"])
		}
	}
	if got := atomic.LoadInt64(&source.calls); got != 0 {
		t.Errorf("source.calls = %d, want 0 (preview)", got)
	}
}

// TestLibraryImportScan_AlreadyImported verifies that a subfolder whose
// absolute path matches an existing series.Path is flagged and does not
// trigger a TVDB lookup.
func TestLibraryImportScan_AlreadyImported(t *testing.T) {
	h := newTestHarness(t)
	h.seedTvdbKey(t)

	dir := t.TempDir()
	for _, sub := range []string{"Already Here", "New Show"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	ctx := context.Background()
	rf, err := h.rootFolder.Create(ctx, dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}
	if _, err := h.series.Create(ctx, library.Series{
		TvdbID: 111, Title: "Already Here", Slug: "already-here",
		Status: "continuing", SeriesType: "standard",
		Path:             filepath.Join(dir, "Already Here"),
		Monitored:        true,
		QualityProfileID: 1,
		SeasonFolder:     true,
		MonitorNewItems:  "all",
	}); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	source := newMockMetadataSource()
	source.results["New Show"] = []metadatasource.SeriesSearchResult{
		{TvdbID: 222, Title: "New Show", Year: 2024, Overview: ""},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId="+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var got []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byName := make(map[string]map[string]any, len(got))
	for _, e := range got {
		byName[e["folderName"].(string)] = e
	}
	already, ok := byName["Already Here"]
	if !ok {
		t.Fatalf("missing Already Here entry")
	}
	if imp, _ := already["alreadyImported"].(bool); !imp {
		t.Errorf("Already Here alreadyImported = false, want true")
	}
	if already["tvdbMatch"] != nil {
		t.Errorf("Already Here tvdbMatch = %v, want nil (skipped lookup)", already["tvdbMatch"])
	}

	newShow, ok := byName["New Show"]
	if !ok {
		t.Fatalf("missing New Show entry")
	}
	if imp, _ := newShow["alreadyImported"].(bool); imp {
		t.Errorf("New Show alreadyImported = true, want false")
	}
	if newShow["tvdbMatch"] == nil {
		t.Errorf("New Show tvdbMatch = nil, want object")
	}

	// Exactly one TVDB lookup (the New Show), because Already Here short-circuited.
	if got := atomic.LoadInt64(&source.calls); got != 1 {
		t.Errorf("source.calls = %d, want 1 (only New Show)", got)
	}
}

// TestLibraryImportScan_ConcurrencyCap seeds 20 subfolders and asserts the
// handler caps in-flight TVDB lookups at 8.
func TestLibraryImportScan_ConcurrencyCap(t *testing.T) {
	h := newTestHarness(t)
	h.seedTvdbKey(t)

	dir := t.TempDir()
	source := newMockMetadataSource()
	source.delay = 20 * time.Millisecond
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("Show %02d", i)
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		source.results[name] = []metadatasource.SeriesSearchResult{
			{TvdbID: int64(1000 + i), Title: name, Year: 2000 + i},
		}
	}
	rf, err := h.rootFolder.Create(context.Background(), dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId="+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if got := atomic.LoadInt64(&source.calls); got != 20 {
		t.Errorf("source.calls = %d, want 20", got)
	}
	source.mu.Lock()
	peak := source.peak
	source.mu.Unlock()
	if peak <= 0 || peak > 8 {
		t.Errorf("peak concurrency = %d, want 1..8", peak)
	}
	t.Logf("observed peak concurrency = %d (cap 8)", peak)
}

// TestLibraryImportScan_UnknownRootFolder verifies that a nonexistent
// rootFolderId returns 404 without walking any filesystem or calling TVDB.
func TestLibraryImportScan_UnknownRootFolder(t *testing.T) {
	h := newTestHarness(t)
	h.seedTvdbKey(t)

	source := newMockMetadataSource()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v3/libraryimport/scan?rootFolderId=99999", nil)
	rr := httptest.NewRecorder()
	h.libraryImportRouter(source).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	if got := atomic.LoadInt64(&source.calls); got != 0 {
		t.Errorf("source.calls = %d, want 0", got)
	}
}

// keys returns the keys of m in unspecified order — test-only helper for
// diagnostic messages.
func keys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
