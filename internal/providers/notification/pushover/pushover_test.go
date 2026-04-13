package pushover

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func pushoverFor(t *testing.T, srv *httptest.Server) *Pushover {
	t.Helper()
	p := New(Settings{UserKey: "testuser", ApiToken: "testtoken"}, srv.Client())
	p.apiURL = srv.URL + "/1/messages.json"
	return p
}

func TestPushoverOnGrab(t *testing.T) {
	var gotForm url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1}`))
	}))
	defer srv.Close()

	p := pushoverFor(t, srv)
	err := p.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Chernobyl",
		EpisodeTitle: "1:23:45",
		Quality:      "HDTV-720p",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotForm.Get("token") != "testtoken" {
		t.Errorf("token: got %q, want testtoken", gotForm.Get("token"))
	}
	if gotForm.Get("user") != "testuser" {
		t.Errorf("user: got %q, want testuser", gotForm.Get("user"))
	}
	if gotForm.Get("title") != "Release Grabbed" {
		t.Errorf("title: got %q, want Release Grabbed", gotForm.Get("title"))
	}
}

func TestPushoverOnDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1}`))
	}))
	defer srv.Close()

	p := pushoverFor(t, srv)
	if err := p.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Chernobyl",
		EpisodeTitle: "Please Remain Calm",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

func TestPushoverOnHealthIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1}`))
	}))
	defer srv.Close()

	p := pushoverFor(t, srv)
	if err := p.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer down",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
}

func TestPushoverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := pushoverFor(t, srv)
	err := p.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}
