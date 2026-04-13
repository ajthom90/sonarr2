package api

import (
	"net/http"
	"time"

	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
)

// KeyAuth returns a middleware that enforces API key authentication.
// Exported so sub-packages (e.g. v6) can use it directly.
func KeyAuth(store hostconfig.Store) func(http.Handler) http.Handler {
	return apiKeyAuth(store)
}

// combinedAuth accepts either a valid API key (X-Api-Key header or ?apikey= param)
// or a valid session cookie. API key is checked first.
func combinedAuth(hcStore hostconfig.Store, sessionStore auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check API key first.
			key := r.Header.Get("X-Api-Key")
			if key == "" {
				key = r.URL.Query().Get("apikey")
			}
			if key != "" {
				hc, err := hcStore.Get(r.Context())
				if err == nil && hc.APIKey == key {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 2. Check session cookie.
			cookie, err := r.Cookie("sonarr2_session")
			if err == nil && cookie.Value != "" {
				session, err := sessionStore.GetByToken(r.Context(), cookie.Value)
				if err == nil && time.Now().Before(session.ExpiresAt) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Neither valid.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
		})
	}
}

// apiKeyAuth enforces API key authentication only. Kept for backward
// compatibility and as a fallback when no SessionStore is configured.
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
