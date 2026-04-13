package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func telegramFor(t *testing.T, srv *httptest.Server) *Telegram {
	t.Helper()
	tg := New(Settings{BotToken: "testtoken", ChatID: "12345"}, srv.Client())
	tg.apiBase = srv.URL
	return tg
}

func TestTelegramOnGrab(t *testing.T) {
	var gotPath string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := telegramFor(t, srv)
	err := tg.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Succession",
		EpisodeTitle: "Celebration",
		Quality:      "Bluray-1080p",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}
	if gotPath != "/bottesttoken/sendMessage" {
		t.Errorf("path: got %q, want /bottesttoken/sendMessage", gotPath)
	}
	if gotBody["chat_id"] != "12345" {
		t.Errorf("chat_id: got %q, want 12345", gotBody["chat_id"])
	}
	if !strings.Contains(gotBody["text"], "Succession") {
		t.Errorf("text should contain series title, got %q", gotBody["text"])
	}
}

func TestTelegramOnDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := telegramFor(t, srv)
	if err := tg.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Succession",
		EpisodeTitle: "Celebration",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

func TestTelegramOnHealthIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := telegramFor(t, srv)
	if err := tg.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Down",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
}

func TestTelegramError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tg := telegramFor(t, srv)
	err := tg.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
