package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/updatecheck"
)

func TestUpdateCheckAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.0.0"})
	}))
	defer srv.Close()

	checker := updatecheck.New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	c := NewUpdateCheck(checker)
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Type != LevelNotice {
		t.Errorf("Type = %q, want notice", results[0].Type)
	}
}

func TestUpdateCheckCurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()

	checker := updatecheck.New("1.0.0", "test", "repo", srv.Client()).WithBaseURL(srv.URL)
	c := NewUpdateCheck(checker)
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}
