package v3

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// rootFolderResource is the Sonarr v3 JSON shape for a root folder.
type rootFolderResource struct {
	ID         int    `json:"id"`
	Path       string `json:"path"`
	FreeSpace  int64  `json:"freeSpace"`
	Unmapped   []any  `json:"unmappedFolders"`
	Accessible bool   `json:"accessible"`
}

// rootFolderHandler handles /api/v3/rootfolder endpoints.
type rootFolderHandler struct {
	series library.SeriesStore
	log    *slog.Logger
}

// NewRootFolderHandler constructs a rootFolderHandler.
func NewRootFolderHandler(series library.SeriesStore, log *slog.Logger) *rootFolderHandler {
	return &rootFolderHandler{series: series, log: log}
}

// MountRootFolder registers /api/v3/rootfolder routes.
func MountRootFolder(r chi.Router, h *rootFolderHandler) {
	r.Route("/api/v3/rootfolder", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

// list handles GET /api/v3/rootfolder.
func (h *rootFolderHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	all, err := h.series.List(ctx)
	if err != nil {
		h.log.Error("rootfolder list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Collect distinct root folders (dir of each series path).
	seen := make(map[string]struct{})
	var roots []string
	for _, s := range all {
		if s.Path == "" {
			continue
		}
		root := filepath.Dir(s.Path)
		if _, ok := seen[root]; !ok {
			seen[root] = struct{}{}
			roots = append(roots, root)
		}
	}

	resources := make([]rootFolderResource, 0, len(roots))
	for i, root := range roots {
		resources = append(resources, rootFolderResource{
			ID:         i + 1,
			Path:       root,
			FreeSpace:  0,
			Unmapped:   []any{},
			Accessible: true,
		})
	}
	writeJSON(w, http.StatusOK, resources)
}
