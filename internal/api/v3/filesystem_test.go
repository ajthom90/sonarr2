package v3_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	v3 "github.com/ajthom90/sonarr2/internal/api/v3"
)

func TestFilesystem_HappyPath(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "tv"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "movies"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644)

	r := chi.NewRouter()
	v3.MountFilesystem(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem?path="+dir, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status: got %d want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Parent      string `json:"parent"`
		Directories []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"directories"`
		Files []any `json:"files"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Directories) != 2 {
		t.Fatalf("want 2 dirs, got %d", len(body.Directories))
	}
	if len(body.Files) != 0 {
		t.Fatalf("files should be empty when includeFiles unset; got %v", body.Files)
	}
}

func TestFilesystem_RejectsTraversal(t *testing.T) {
	r := chi.NewRouter()
	v3.MountFilesystem(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem?path=/data/../etc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400", w.Code)
	}
}

func TestFilesystem_MissingPath(t *testing.T) {
	r := chi.NewRouter()
	v3.MountFilesystem(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400", w.Code)
	}
}

func TestFilesystem_DenyList(t *testing.T) {
	r := chi.NewRouter()
	v3.MountFilesystem(r)
	for _, p := range []string{"/proc", "/sys", "/dev"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem?path="+p, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Errorf("%s: status got %d want 403", p, w.Code)
		}
	}
}

func TestFilesystem_NotFound(t *testing.T) {
	r := chi.NewRouter()
	v3.MountFilesystem(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem?path=/nope/nope/definitely-not-here", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("status: got %d want 404", w.Code)
	}
}

func TestFilesystem_IncludeFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "show.txt"), []byte(""), 0o644)
	r := chi.NewRouter()
	v3.MountFilesystem(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v3/filesystem?path="+dir+"&includeFiles=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Files []any `json:"files"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Files) != 1 {
		t.Fatalf("want 1 file, got %d", len(body.Files))
	}
}
