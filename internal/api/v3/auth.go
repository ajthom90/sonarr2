package v3

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/go-chi/chi/v5"
)

// MountAuth registers authentication endpoints. These are OUTSIDE the auth
// middleware group — they must be accessible without authentication.
func MountAuth(r chi.Router, userStore auth.UserStore, sessionStore auth.SessionStore, hcStore hostconfig.Store) {
	r.Get("/api/v3/initialize", handleInitializeCheck(userStore))
	r.Post("/api/v3/initialize", handleInitialize(userStore, hcStore))
	r.Post("/api/v3/login", handleLogin(userStore, sessionStore))
	r.Post("/api/v3/logout", handleLogout(sessionStore))
}

// GET /api/v3/initialize — returns {"initialized": true/false}
func handleInitializeCheck(userStore auth.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := userStore.Count(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"initialized": count > 0})
	}
}

// POST /api/v3/initialize — creates the first user. Only works when no users exist.
// Body: {"username": "admin", "password": "secret"}
func handleInitialize(userStore auth.UserStore, hcStore hostconfig.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := userStore.Count(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		if count > 0 {
			writeJSON(w, http.StatusConflict, map[string]string{"message": "Already initialized"})
			return
		}

		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
			return
		}
		if body.Username == "" || body.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Username and password are required"})
			return
		}

		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to hash password"})
			return
		}

		user, err := userStore.Create(r.Context(), body.Username, hash)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		// Also update auth mode to "forms" now that we have a user.
		hc, _ := hcStore.Get(r.Context())
		hc.AuthMode = "forms"
		_ = hcStore.Upsert(r.Context(), hc)

		writeJSON(w, http.StatusCreated, map[string]any{
			"username": user.Username,
			"apiKey": func() string {
				hc2, _ := hcStore.Get(r.Context())
				return hc2.APIKey
			}(),
		})
	}
}

// POST /api/v3/login — validates credentials and returns a session cookie.
// Body: {"username": "admin", "password": "secret"}
func handleLogin(userStore auth.UserStore, sessionStore auth.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
			return
		}

		user, err := userStore.GetByUsername(r.Context(), body.Username)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid credentials"})
			return
		}

		if !auth.CheckPassword(user.PasswordHash, body.Password) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid credentials"})
			return
		}

		// Create session.
		token := auth.NewSessionToken()
		session := auth.Session{
			Token:     token,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(auth.SessionTTL),
		}
		if err := sessionStore.Create(r.Context(), session); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create session"})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "sonarr2_session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(auth.SessionTTL.Seconds()),
		})

		writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
	}
}

// POST /api/v3/logout — clears the session cookie and deletes the session.
func handleLogout(sessionStore auth.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sonarr2_session")
		if err == nil && cookie.Value != "" {
			_ = sessionStore.DeleteByToken(r.Context(), cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "sonarr2_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1, // delete cookie
		})

		w.WriteHeader(http.StatusOK)
	}
}
