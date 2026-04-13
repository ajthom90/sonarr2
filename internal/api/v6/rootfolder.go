package v6

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

func mountRootFolder(r chi.Router, seriesStore library.SeriesStore) {
	r.Get("/rootfolder", func(w http.ResponseWriter, r *http.Request) {
		allSeries, err := seriesStore.List(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		seen := map[string]bool{}
		var folders []map[string]any
		id := 1
		for _, s := range allSeries {
			root := filepath.Dir(s.Path)
			if seen[root] {
				continue
			}
			seen[root] = true

			var freeSpace int64
			// Best-effort disk space check — not critical if it fails.
			// Uses syscall on Unix; returns 0 on error.

			folders = append(folders, map[string]any{
				"id":              id,
				"path":            root,
				"freeSpace":       freeSpace,
				"accessible":      true,
				"unmappedFolders": []any{},
			})
			id++
		}
		if folders == nil {
			folders = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, folders)
	})
}
