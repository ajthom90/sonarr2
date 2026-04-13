package api

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/hostconfig"
)

func apiKeyAuth(store hostconfig.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Api-Key")
			if key == "" {
				key = r.URL.Query().Get("apikey")
			}
			if key == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
				return
			}
			hc, err := store.Get(r.Context())
			if err != nil || hc.APIKey != key {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
