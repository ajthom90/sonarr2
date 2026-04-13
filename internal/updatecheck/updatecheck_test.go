package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCheckUpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.0.0"})
	}))
	defer srv.Close()

	c := New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	result, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.UpdateAvailable {
		t.Error("expected update available")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want 2.0.0", result.LatestVersion)
	}
}

func TestCheckNoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()

	c := New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	result, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.UpdateAvailable {
		t.Error("expected no update")
	}
}

func TestCheckDevVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.0.0"})
	}))
	defer srv.Close()

	c := New("dev", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	result, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.UpdateAvailable {
		t.Error("dev version should not report updates")
	}
}

func TestCheckCachesResult(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()

	c := New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	c.Check(context.Background())
	c.Check(context.Background())

	if calls.Load() != 1 {
		t.Errorf("API called %d times, want 1 (cached)", calls.Load())
	}
}

func TestCheckAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	result, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("should not error on non-200: %v", err)
	}
	if result.UpdateAvailable {
		t.Error("should not report update on API error")
	}
}
