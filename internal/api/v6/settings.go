package v6

import (
	"encoding/json"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/go-chi/chi/v5"
)

func mountSettings(r chi.Router, store hostconfig.Store, onTvdbKeyChanged func(string)) {
	r.Route("/config/general", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			hc, err := store.Get(r.Context())
			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"apiKey":     hc.APIKey,
				"authMode":   hc.AuthMode,
				"tvdbApiKey": hc.TvdbApiKey,
			})
		})

		r.Put("/", func(w http.ResponseWriter, r *http.Request) {
			hc, err := store.Get(r.Context())
			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, err.Error())
				return
			}

			var body struct {
				TvdbApiKey *string `json:"tvdbApiKey"`
				AuthMode   *string `json:"authMode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				WriteBadRequest(w, r, "Invalid request body")
				return
			}

			tvdbKeyChanged := false
			if body.TvdbApiKey != nil {
				hc.TvdbApiKey = *body.TvdbApiKey
				tvdbKeyChanged = true
			}
			if body.AuthMode != nil {
				hc.AuthMode = *body.AuthMode
			}

			if err := store.Upsert(r.Context(), hc); err != nil {
				WriteError(w, r, http.StatusInternalServerError, err.Error())
				return
			}

			if tvdbKeyChanged && onTvdbKeyChanged != nil {
				onTvdbKeyChanged(hc.TvdbApiKey)
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"apiKey":     hc.APIKey,
				"authMode":   hc.AuthMode,
				"tvdbApiKey": hc.TvdbApiKey,
			})
		})
	})
}
