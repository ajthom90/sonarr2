package v6

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/go-chi/chi/v5"
)

func mountHealth(r chi.Router, checker *health.Checker) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		results := checker.Results()
		if results == nil {
			results = []health.Result{}
		}
		writeJSON(w, http.StatusOK, results)
	})
}
