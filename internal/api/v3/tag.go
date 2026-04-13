package v3

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountTag registers /api/v3/tag routes.
// Returns an empty array (stub — real tag management is M18+).
func MountTag(r chi.Router) {
	r.Route("/api/v3/tag", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, []any{})
		})
	})
}
