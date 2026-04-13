package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func discordFor(t *testing.T, srv *httptest.Server) *Discord {
	t.Helper()
	return New(Settings{WebhookURL: srv.URL + "/webhook"}, srv.Client())
}

func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return m
}

func TestDiscordOnGrab(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = readBody(t, r)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := discordFor(t, srv)
	err := d.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Breaking Bad",
		EpisodeTitle: "Pilot",
		Quality:      "HDTV-720p",
		Indexer:      "MyIndexer",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %q, want POST", gotMethod)
	}
	if gotPath != "/webhook" {
		t.Errorf("path: got %q, want /webhook", gotPath)
	}

	embeds, ok := gotBody["embeds"].([]any)
	if !ok || len(embeds) == 0 {
		t.Fatalf("expected embeds array in payload")
	}
	embed := embeds[0].(map[string]any)
	if embed["title"] != "Release Grabbed" {
		t.Errorf("embed title: got %v, want Release Grabbed", embed["title"])
	}
}

func TestDiscordOnDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := discordFor(t, srv)
	if err := d.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Breaking Bad",
		EpisodeTitle: "Pilot",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

func TestDiscordOnHealthIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := discordFor(t, srv)
	if err := d.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer is offline",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
}

func TestDiscordWebhookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := discordFor(t, srv)
	err := d.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
