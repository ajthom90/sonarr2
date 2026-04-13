package v3

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/go-chi/chi/v5"
)

// MountHealth registers /api/v3/health routes.
func MountHealth(r chi.Router, checker *health.Checker) {
	r.Route("/api/v3/health", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var results []health.Result
			if checker != nil {
				results = checker.Results()
			}
			if results == nil {
				results = []health.Result{}
			}
			writeJSON(w, http.StatusOK, results)
		})
	})
}
