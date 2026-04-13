package slack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func slackFor(t *testing.T, srv *httptest.Server) *Slack {
	t.Helper()
	return New(Settings{WebhookURL: srv.URL + "/webhook", Channel: "#sonarr"}, srv.Client())
}

func TestSlackOnGrab(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := slackFor(t, srv)
	err := s.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "The Wire",
		EpisodeTitle: "The Target",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotBody["channel"] != "#sonarr" {
		t.Errorf("channel: got %v, want #sonarr", gotBody["channel"])
	}
	attachments, ok := gotBody["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		t.Fatalf("expected attachments in payload")
	}
	att := attachments[0].(map[string]any)
	if att["color"] != "good" {
		t.Errorf("color: got %v, want good", att["color"])
	}
}

func TestSlackOnDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := slackFor(t, srv)
	if err := s.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "The Wire",
		EpisodeTitle: "The Target",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

func TestSlackOnHealthIssue(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := slackFor(t, srv)
	if err := s.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer is offline",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
	attachments, ok := gotBody["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		t.Fatalf("expected attachments")
	}
	att := attachments[0].(map[string]any)
	if att["color"] != "danger" {
		t.Errorf("color: got %v, want danger", att["color"])
	}
}

func TestSlackWebhookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := slackFor(t, srv)
	err := s.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
