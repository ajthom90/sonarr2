package v3

import (
	"encoding/json"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/go-chi/chi/v5"
)

// MountSettings registers /api/v3/config/general routes.
func MountSettings(r chi.Router, store hostconfig.Store, onTvdbKeyChanged func(string)) {
	r.Route("/api/v3/config/general", func(r chi.Router) {
		r.Get("/", handleGetSettings(store))
		r.Put("/", handleUpdateSettings(store, onTvdbKeyChanged))
	})
}

func handleGetSettings(store hostconfig.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hc, err := store.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"apiKey":     hc.APIKey,
			"authMode":   hc.AuthMode,
			"tvdbApiKey": hc.TvdbApiKey,
		})
	}
}

func handleUpdateSettings(store hostconfig.Store, onTvdbKeyChanged func(string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hc, err := store.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		var body struct {
			TvdbApiKey *string `json:"tvdbApiKey"`
			AuthMode   *string `json:"authMode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
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
	}
}
