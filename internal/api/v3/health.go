package v3

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountHealth registers /api/v3/health routes.
// Returns an empty array (stub — real health checks are M18+).
func MountHealth(r chi.Router) {
	r.Route("/api/v3/health", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, []any{})
		})
	})
}
