package gotify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func gotifyFor(t *testing.T, srv *httptest.Server) *Gotify {
	t.Helper()
	return New(Settings{ServerURL: srv.URL, AppToken: "testapptoken"}, srv.Client())
}

func TestGotifyOnGrab(t *testing.T) {
	var gotPath, gotToken string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Gotify-Key")
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := gotifyFor(t, srv)
	err := g.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Dark",
		EpisodeTitle: "Secrets",
		Quality:      "HDTV-720p",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotPath != "/message" {
		t.Errorf("path: got %q, want /message", gotPath)
	}
	if gotToken != "testapptoken" {
		t.Errorf("X-Gotify-Key: got %q, want testapptoken", gotToken)
	}
	if gotBody["title"] != "Release Grabbed" {
		t.Errorf("title: got %v, want Release Grabbed", gotBody["title"])
	}
	if gotBody["priority"] != float64(5) {
		t.Errorf("priority: got %v, want 5", gotBody["priority"])
	}
}

func TestGotifyOnDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := gotifyFor(t, srv)
	if err := g.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Dark",
		EpisodeTitle: "Secrets",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

func TestGotifyOnHealthIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := gotifyFor(t, srv)
	if err := g.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer down",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
}

func TestGotifyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	g := gotifyFor(t, srv)
	err := g.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
