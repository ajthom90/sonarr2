package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func webhookFor(t *testing.T, srv *httptest.Server) *Webhook {
	t.Helper()
	return New(Settings{URL: srv.URL + "/hook", Method: "POST"}, srv.Client())
}

func TestWebhookOnGrab(t *testing.T) {
	var gotBody map[string]any
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := webhookFor(t, srv)
	err := wh.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Lost",
		EpisodeTitle: "Pilot",
		Quality:      "HDTV-720p",
		Indexer:      "TestIndexer",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %q, want POST", gotMethod)
	}
	if gotBody["eventType"] != "Grab" {
		t.Errorf("eventType: got %v, want Grab", gotBody["eventType"])
	}
	if gotBody["seriesTitle"] != "Lost" {
		t.Errorf("seriesTitle: got %v, want Lost", gotBody["seriesTitle"])
	}
}

func TestWebhookOnDownload(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := webhookFor(t, srv)
	if err := wh.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Lost",
		EpisodeTitle: "Pilot",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
	if gotBody["eventType"] != "Download" {
		t.Errorf("eventType: got %v, want Download", gotBody["eventType"])
	}
}

func TestWebhookOnHealthIssue(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := webhookFor(t, srv)
	if err := wh.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer is offline",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
	if gotBody["eventType"] != "HealthIssue" {
		t.Errorf("eventType: got %v, want HealthIssue", gotBody["eventType"])
	}
}

func TestWebhookDefaultMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Method is empty — should default to POST.
	wh := New(Settings{URL: srv.URL + "/hook"}, srv.Client())
	_ = wh.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if gotMethod != http.MethodPost {
		t.Errorf("default method: got %q, want POST", gotMethod)
	}
}

func TestWebhookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	wh := webhookFor(t, srv)
	err := wh.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
