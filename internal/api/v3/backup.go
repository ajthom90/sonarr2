package v3

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/backup"
	"github.com/go-chi/chi/v5"
)

// MountBackup registers /api/v3/system/backup routes.
func MountBackup(r chi.Router, svc *backup.Service) {
	r.Route("/api/v3/system/backup", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			list, err := svc.List(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			info, err := svc.Create(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, info)
		})

		r.Get("/{name}", func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "name")
			// Sanitize: only allow filenames, no path separators
			if strings.ContainsAny(name, "/\\") || name != filepath.Base(name) {
				http.Error(w, "invalid backup name", http.StatusBadRequest)
				return
			}
			path, err := svc.FilePath(name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Disposition", "attachment; filename="+name)
			http.ServeFile(w, r, path)
		})

		r.Delete("/{name}", func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "name")
			if strings.ContainsAny(name, "/\\") || name != filepath.Base(name) {
				http.Error(w, "invalid backup name", http.StatusBadRequest)
				return
			}
			if err := svc.Delete(r.Context(), name); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			w.WriteHeader(http.StatusOK)
		})
	})
}
