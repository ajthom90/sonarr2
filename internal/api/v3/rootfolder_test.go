package v3_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	v3 "github.com/ajthom90/sonarr2/internal/api/v3"
	"github.com/ajthom90/sonarr2/internal/library"
)

// rootFolderRouter mounts only the rootfolder endpoints against h's stores.
func (h *testHarness) rootFolderRouter() chi.Router {
	r := chi.NewRouter()
	v3.MountRootFolder(r, h.rootFolder, h.series)
	return r
}

func TestRootFolder_PostCreatesRow(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()

	body := `{"path":"` + dir + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/rootfolder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.rootFolderRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id, ok := got["id"].(float64); !ok || id == 0 {
		t.Errorf("id = %v, want non-zero number", got["id"])
	}
	if got["path"] != dir {
		t.Errorf("path = %v, want %q", got["path"], dir)
	}
	for _, field := range []string{"id", "path", "freeSpace", "accessible", "unmappedFolders"} {
		if _, ok := got[field]; !ok {
			t.Errorf("missing field %q", field)
		}
	}
}

func TestRootFolder_PostRejectsNonExistent(t *testing.T) {
	h := newTestHarness(t)

	body := `{"path":"/nope/this/path/does/not/exist"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v3/rootfolder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.rootFolderRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
}

func TestRootFolder_PostRejectsDuplicate(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()

	body := `{"path":"` + dir + `"}`
	for i, want := range []int{http.StatusCreated, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/api/v3/rootfolder", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.rootFolderRouter().ServeHTTP(rr, req)
		if rr.Code != want {
			t.Fatalf("attempt %d: status = %d, want %d; body = %s",
				i+1, rr.Code, want, rr.Body.String())
		}
	}
}

func TestRootFolder_GetListsPersisted(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()

	if _, err := h.rootFolder.Create(context.Background(), dir); err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v3/rootfolder", nil)
	rr := httptest.NewRecorder()
	h.rootFolderRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var list []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d rows, want 1", len(list))
	}
	if list[0]["path"] != dir {
		t.Errorf("path = %v, want %q", list[0]["path"], dir)
	}
	if accessible, _ := list[0]["accessible"].(bool); !accessible {
		t.Errorf("accessible = %v, want true", list[0]["accessible"])
	}
}

func TestRootFolder_DeleteBlockedWhenSeriesReferences(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()
	ctx := context.Background()

	rf, err := h.rootFolder.Create(ctx, dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}
	// Create a series whose path sits directly under the root folder.
	if _, err := h.series.Create(ctx, library.Series{
		TvdbID: 9001, Title: "Blocker Show", Slug: "blocker-show",
		Status: "continuing", SeriesType: "standard",
		Path: dir + "/Blocker Show", Monitored: true,
		QualityProfileID: 1, SeasonFolder: true, MonitorNewItems: "all",
	}); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v3/rootfolder/"+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.rootFolderRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	titles, ok := body["affectedSeries"].([]any)
	if !ok {
		t.Fatalf("affectedSeries missing or wrong type: %v", body)
	}
	if len(titles) != 1 || titles[0] != "Blocker Show" {
		t.Errorf("affectedSeries = %v, want [Blocker Show]", titles)
	}
}

func TestRootFolder_DeleteUnused(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()
	ctx := context.Background()

	rf, err := h.rootFolder.Create(ctx, dir)
	if err != nil {
		t.Fatalf("seed rootfolder: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v3/rootfolder/"+strconv.FormatInt(rf.ID, 10), nil)
	rr := httptest.NewRecorder()
	h.rootFolderRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rr.Code, rr.Body.String())
	}

	// Verify the row is gone.
	if _, err := h.rootFolder.Get(ctx, rf.ID); err == nil {
		t.Errorf("rootfolder still present after delete")
	}
}
