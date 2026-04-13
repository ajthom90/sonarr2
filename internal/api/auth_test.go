package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/hostconfig"
)

type stubHostConfig struct {
	apiKey string
}

func (s *stubHostConfig) Get(ctx context.Context) (hostconfig.HostConfig, error) {
	return hostconfig.HostConfig{APIKey: s.apiKey}, nil
}

func (s *stubHostConfig) Upsert(ctx context.Context, hc hostconfig.HostConfig) error {
	return nil
}

// okHandler is a trivial handler that returns 200 OK; used to sit behind the
// auth middleware in tests.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthValidKey(t *testing.T) {
	store := &stubHostConfig{apiKey: "testkey"}
	handler := apiKeyAuth(store)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Api-Key", "testkey")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestAuthMissingKey(t *testing.T) {
	store := &stubHostConfig{apiKey: "testkey"}
	handler := apiKeyAuth(store)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthWrongKey(t *testing.T) {
	store := &stubHostConfig{apiKey: "testkey"}
	handler := apiKeyAuth(store)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Api-Key", "badkey")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthQueryParam(t *testing.T) {
	store := &stubHostConfig{apiKey: "testkey"}
	handler := apiKeyAuth(store)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/?apikey=testkey", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
