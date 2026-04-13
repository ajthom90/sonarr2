package v3

import (
	"bytes"
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

func buildEpisodeRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := NewEpisodeHandler(lib.Episodes, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountEpisode(r, h)
	return r
}

func buildEpisodeFileRouter(lib *library.Library) http.Handler {
	r := chi.NewRouter()
	h := NewEpisodeFileHandler(lib.EpisodeFiles, lib.Series, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	MountEpisodeFile(r, h)
	return r
}

func TestEpisodeListBySeries(t *testing.T) {
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
	ep, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Pilot",
		Overview:      "First episode",
		AirDateUtc:    &airDate,
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode: %v", err)
	}

	router := buildEpisodeRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/episode?seriesId="+itoa(series.ID), nil)
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
		t.Fatalf("got %d episodes, want 1", len(result))
	}

	ep0 := result[0]
	if ep0["id"].(float64) != float64(ep.ID) {
		t.Errorf("id = %v, want %v", ep0["id"], ep.ID)
	}
	if ep0["seriesId"].(float64) != float64(series.ID) {
		t.Errorf("seriesId = %v, want %v", ep0["seriesId"], series.ID)
	}
	if ep0["title"] != "Pilot" {
		t.Errorf("title = %v, want Pilot", ep0["title"])
	}
	if ep0["airDate"] != "2024-01-15" {
		t.Errorf("airDate = %v, want 2024-01-15", ep0["airDate"])
	}
	if ep0["airDateUtc"] != "2024-01-15T05:00:00Z" {
		t.Errorf("airDateUtc = %v, want 2024-01-15T05:00:00Z", ep0["airDateUtc"])
	}
	if ep0["hasFile"] != false {
		t.Errorf("hasFile = %v, want false", ep0["hasFile"])
	}
	if ep0["monitored"] != true {
		t.Errorf("monitored = %v, want true", ep0["monitored"])
	}

	// Verify required field names are present.
	for _, field := range []string{"id", "seriesId", "tvdbId", "episodeFileId", "seasonNumber",
		"episodeNumber", "title", "airDate", "airDateUtc", "overview", "hasFile",
		"monitored", "runtime", "unverifiedSceneNumbering"} {
		if _, ok := ep0[field]; !ok {
			t.Errorf("missing field %q in episode response", field)
		}
	}
}

func TestEpisodeGetByID(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 2, Title: "Show 2", Slug: "show-2",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Show 2", Monitored: true,
	})

	ep, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 2,
		Title:         "Episode 2",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode: %v", err)
	}

	router := buildEpisodeRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/episode/"+itoa(ep.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["title"] != "Episode 2" {
		t.Errorf("title = %v, want Episode 2", result["title"])
	}
}

func TestEpisodeGetNotFound(t *testing.T) {
	lib := setupLibrary(t)
	router := buildEpisodeRouter(lib)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/episode/99999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestEpisodeUpdateMonitored(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 3, Title: "Show 3", Slug: "show-3",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Show 3", Monitored: true,
	})

	ep, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Pilot",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("Create episode: %v", err)
	}

	router := buildEpisodeRouter(lib)
	body := `{"monitored":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v3/episode/"+itoa(ep.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["monitored"] != false {
		t.Errorf("monitored = %v, want false", result["monitored"])
	}
}

func TestEpisodeListMissingSeriesID(t *testing.T) {
	lib := setupLibrary(t)
	router := buildEpisodeRouter(lib)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/episode", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestEpisodeFileList(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID: 10, Title: "File Show", Slug: "file-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/File Show", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	f, err := lib.EpisodeFiles.Create(ctx, library.EpisodeFile{
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
	req := httptest.NewRequest(http.MethodGet, "/api/v3/episodefile?seriesId="+itoa(series.ID), nil)
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

	f0 := result[0]
	if f0["id"].(float64) != float64(f.ID) {
		t.Errorf("id = %v, want %v", f0["id"], f.ID)
	}
	if f0["relativePath"] != "Season 01/File.Show.S01E01.mkv" {
		t.Errorf("relativePath = %v", f0["relativePath"])
	}
	if f0["path"] != "/tv/File Show/Season 01/File.Show.S01E01.mkv" {
		t.Errorf("path = %v", f0["path"])
	}
	if f0["releaseGroup"] != "GROUP" {
		t.Errorf("releaseGroup = %v", f0["releaseGroup"])
	}

	// Verify required field names.
	for _, field := range []string{"id", "seriesId", "seasonNumber", "relativePath",
		"path", "size", "dateAdded", "quality", "releaseGroup"} {
		if _, ok := f0[field]; !ok {
			t.Errorf("missing field %q in episodefile response", field)
		}
	}
}

func TestEpisodeFileGetByID(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 11, Title: "Show 11", Slug: "show-11",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Show 11", Monitored: true,
	})

	f, err := lib.EpisodeFiles.Create(ctx, library.EpisodeFile{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		RelativePath: "Season 01/ep1.mkv",
		Size:         500000000,
		DateAdded:    time.Now(),
		QualityName:  "WEB-1080p",
	})
	if err != nil {
		t.Fatalf("Create episode file: %v", err)
	}

	router := buildEpisodeFileRouter(lib)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/episodefile/"+itoa(f.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["relativePath"] != "Season 01/ep1.mkv" {
		t.Errorf("relativePath = %v", result["relativePath"])
	}
}

func TestEpisodeFileDelete(t *testing.T) {
	lib := setupLibrary(t)
	ctx := context.Background()

	series, _ := lib.Series.Create(ctx, library.Series{
		TvdbID: 12, Title: "Show 12", Slug: "show-12",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Show 12", Monitored: true,
	})

	f, err := lib.EpisodeFiles.Create(ctx, library.EpisodeFile{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		RelativePath: "Season 01/ep2.mkv",
		Size:         100000000,
		DateAdded:    time.Now(),
		QualityName:  "HDTV-720p",
	})
	if err != nil {
		t.Fatalf("Create episode file: %v", err)
	}

	router := buildEpisodeFileRouter(lib)

	// DELETE.
	req := httptest.NewRequest(http.MethodDelete, "/api/v3/episodefile/"+itoa(f.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", rr.Code)
	}

	// GET should return 404.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v3/episodefile/"+itoa(f.ID), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want 404", rr2.Code)
	}
}
