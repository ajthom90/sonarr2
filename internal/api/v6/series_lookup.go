package v6

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

func mountSeriesLookup(r chi.Router, source metadatasource.MetadataSource) {
	r.Get("/series/lookup", func(w http.ResponseWriter, r *http.Request) {
		term := r.URL.Query().Get("term")
		if term == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "term parameter is required"})
			return
		}

		results, err := source.SearchSeries(r.Context(), term)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

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
	})
}
