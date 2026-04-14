package v3

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// filesystemEntry is one row in the directory/file list.
type filesystemEntry struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// filesystemResponse is the Sonarr-compat wire shape for GET /api/v3/filesystem.
type filesystemResponse struct {
	Parent      string            `json:"parent"`
	Directories []filesystemEntry `json:"directories"`
	Files       []filesystemEntry `json:"files"`
}

// denyListedPrefixes lists absolute paths we refuse to browse regardless of
// filesystem permissions. These are read-only system trees that don't help
// users organize TV libraries and leak unnecessary info about the host.
var denyListedPrefixes = []string{"/proc", "/sys", "/dev", "/.git"}

// MountFilesystem registers GET /api/v3/filesystem on r.
func MountFilesystem(r chi.Router) {
	r.Get("/api/v3/filesystem", handleFilesystem)
}

func handleFilesystem(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}

	cleaned := filepath.Clean(path)
	if cleaned != path || strings.Contains(path, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if isDenyListed(cleaned) {
		writeError(w, http.StatusForbidden, "path is not browsable")
		return
	}

	includeFiles := r.URL.Query().Get("includeFiles") == "true"

	info, err := os.Stat(cleaned)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}
	if errors.Is(err, os.ErrPermission) {
		writeError(w, http.StatusForbidden, "permission denied")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a directory")
		return
	}

	entries, err := os.ReadDir(cleaned)
	if errors.Is(err, os.ErrPermission) {
		writeError(w, http.StatusForbidden, "permission denied")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dirs := []filesystemEntry{}
	files := []filesystemEntry{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(cleaned, e.Name())
		if e.IsDir() {
			dirs = append(dirs, filesystemEntry{Type: "folder", Name: e.Name(), Path: full})
		} else if includeFiles {
			files = append(files, filesystemEntry{Type: "file", Name: e.Name(), Path: full})
		}
	}

	resp := filesystemResponse{
		Parent:      filepath.Dir(cleaned),
		Directories: dirs,
		Files:       files,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func isDenyListed(path string) bool {
	for _, p := range denyListedPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
