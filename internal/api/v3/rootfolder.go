package v3

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

// rootFolderResource is the Sonarr v3 JSON shape for a root folder.
type rootFolderResource struct {
	ID              int64  `json:"id"`
	Path            string `json:"path"`
	FreeSpace       int64  `json:"freeSpace"`
	Accessible      bool   `json:"accessible"`
	UnmappedFolders []any  `json:"unmappedFolders"`
}

// rootFolderHandler handles /api/v3/rootfolder endpoints backed by a
// persistent rootfolder.Store. The library.SeriesStore is consulted on
// DELETE so we can reject removals that would orphan series rows.
type rootFolderHandler struct {
	rf     rootfolder.Store
	series library.SeriesStore
	log    *slog.Logger
}

// NewRootFolderHandler constructs a rootFolderHandler.
func NewRootFolderHandler(rf rootfolder.Store, series library.SeriesStore, log *slog.Logger) *rootFolderHandler {
	return &rootFolderHandler{rf: rf, series: series, log: log}
}

// MountRootFolder registers /api/v3/rootfolder CRUD routes.
func MountRootFolder(r chi.Router, rf rootfolder.Store, series library.SeriesStore) {
	h := NewRootFolderHandler(rf, series, slog.Default())
	r.Route("/api/v3/rootfolder", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Delete("/{id}", h.delete)
	})
}

// list handles GET /api/v3/rootfolder.
func (h *rootFolderHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.rf.List(r.Context())
	if err != nil {
		h.log.Error("rootfolder list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]rootFolderResource, 0, len(rows))
	for _, rr := range rows {
		out = append(out, toRootFolderResource(rr))
	}
	writeJSON(w, http.StatusOK, out)
}

// create handles POST /api/v3/rootfolder.
func (h *rootFolderHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	cleaned := filepath.Clean(body.Path)
	info, err := os.Stat(cleaned)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusBadRequest, "folder does not exist: "+cleaned)
		return
	}
	if errors.Is(err, os.ErrPermission) {
		writeError(w, http.StatusForbidden, "permission denied")
		return
	}
	if err != nil {
		h.log.Error("rootfolder stat", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a directory")
		return
	}
	rf, err := h.rf.Create(r.Context(), cleaned)
	if errors.Is(err, rootfolder.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "root folder already exists")
		return
	}
	if err != nil {
		h.log.Error("rootfolder create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toRootFolderResource(rf))
}

// delete handles DELETE /api/v3/rootfolder/{id}. Returns 409 (with an
// affectedSeries list) if any series row still lives under this root.
func (h *rootFolderHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	target, err := h.rf.Get(r.Context(), id)
	if errors.Is(err, rootfolder.ErrNotFound) {
		writeError(w, http.StatusNotFound, "root folder not found")
		return
	}
	if err != nil {
		h.log.Error("rootfolder get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	affected, err := h.affectedSeriesTitles(r.Context(), target.Path)
	if err != nil {
		h.log.Error("rootfolder affected series", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if len(affected) > 0 {
		writeJSON(w, http.StatusConflict, map[string]any{
			"message":        "root folder is still referenced by series; reassign or remove them first",
			"affectedSeries": affected,
		})
		return
	}
	if err := h.rf.Delete(r.Context(), id); err != nil {
		h.log.Error("rootfolder delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// affectedSeriesTitles returns up to 5 series titles that sit directly
// under root (i.e. filepath.Dir(series.Path) == root). This drives the
// 409 response body on DELETE.
func (h *rootFolderHandler) affectedSeriesTitles(ctx context.Context, root string) ([]string, error) {
	if h.series == nil {
		return nil, nil
	}
	all, err := h.series.List(ctx)
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, 5)
	for _, s := range all {
		if s.Path == "" {
			continue
		}
		if filepath.Dir(s.Path) == root {
			titles = append(titles, s.Title)
			if len(titles) == 5 {
				break
			}
		}
	}
	return titles, nil
}

// toRootFolderResource maps a domain RootFolder to the wire resource,
// filling in live stat-backed fields (freeSpace, accessible).
func toRootFolderResource(rf rootfolder.RootFolder) rootFolderResource {
	return rootFolderResource{
		ID:              rf.ID,
		Path:            rf.Path,
		FreeSpace:       freeSpaceBytes(rf.Path),
		Accessible:      isAccessible(rf.Path),
		UnmappedFolders: []any{},
	}
}

// freeSpaceBytes returns the available bytes on the filesystem backing
// path. Returns 0 on any error so the client still gets a usable payload.
func freeSpaceBytes(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	return int64(stat.Bavail) * int64(stat.Bsize)
}

// isAccessible returns true if the process can open path for reading.
// A closed best-effort probe — callers should not treat this as a
// security check.
func isAccessible(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
