package v3

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// MountSeriesLookup registers the /api/v3/series/lookup route on r.
func MountSeriesLookup(r chi.Router, source metadatasource.MetadataSource) {
	r.Get("/api/v3/series/lookup", handleSeriesLookup(source))
}

func handleSeriesLookup(source metadatasource.MetadataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term := r.URL.Query().Get("term")
		if term == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "term parameter is required"})
			return
		}

		results, err := source.SearchSeries(r.Context(), term)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "401") || strings.Contains(msg, "login") {
				msg = "TVDB API key is not configured. Set it in Settings → General."
			}
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"message": msg})
			return
		}

		// Transform to Sonarr-compatible response format.
		var response []map[string]any
		for _, r := range results {
			response = append(response, map[string]any{
				"tvdbId":    r.TvdbID,
				"title":     r.Title,
				"year":      r.Year,
				"overview":  r.Overview,
				"status":    r.Status,
				"network":   r.Network,
				"titleSlug": r.Slug,
			})
		}
		if response == nil {
			response = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, response)
	}
}
