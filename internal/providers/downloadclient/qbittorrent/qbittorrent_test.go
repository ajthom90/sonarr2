package qbittorrent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// qbtFor builds a QBittorrent client pointed at the given httptest.Server.
func qbtFor(t *testing.T, srv *httptest.Server) *QBittorrent {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return New(Settings{
		Host:     u.Hostname(),
		Port:     port,
		Username: "admin",
		Password: "password",
		Category: "tv",
	}, srv.Client())
}

// loginOK writes the successful login response.
func loginOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Ok."))
}

func TestQBittorrentAdd(t *testing.T) {
	var addCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			loginOK(w)
		case "/api/v2/torrents/add":
			addCalled = true
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.FormValue("urls") != "magnet:?xt=urn:test" {
				t.Errorf("urls: got %q", r.FormValue("urls"))
			}
			if r.FormValue("category") != "tv" {
				t.Errorf("category: got %q", r.FormValue("category"))
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	qbt := qbtFor(t, srv)
	id, err := qbt.Add(context.Background(), "magnet:?xt=urn:test", "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "magnet:?xt=urn:test" {
		t.Errorf("Add: got ID %q", id)
	}
	if !addCalled {
		t.Error("torrents/add was not called")
	}
}

func TestQBittorrentItems(t *testing.T) {
	const cannedTorrents = `[{"hash":"abc123","name":"Show.S01E01","state":"downloading","size":1000000000,"amount_left":400000000,"save_path":"/downloads/tv"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			loginOK(w)
		case "/api/v2/torrents/info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(cannedTorrents))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	qbt := qbtFor(t, srv)
	items, err := qbt.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.DownloadID != "abc123" {
		t.Errorf("DownloadID: got %q, want abc123", item.DownloadID)
	}
	if item.Title != "Show.S01E01" {
		t.Errorf("Title: got %q", item.Title)
	}
	if item.TotalSize != 1000000000 {
		t.Errorf("TotalSize: got %d", item.TotalSize)
	}
	if item.Remaining != 400000000 {
		t.Errorf("Remaining: got %d", item.Remaining)
	}
}

func TestQBittorrentTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			loginOK(w)
		case "/api/v2/app/version":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("v5.0.0"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	qbt := qbtFor(t, srv)
	if err := qbt.Test(context.Background()); err != nil {
		t.Errorf("Test() returned error: %v", err)
	}
}

func TestQBittorrentLoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Fails."))
		}
	}))
	defer srv.Close()

	qbt := qbtFor(t, srv)
	if err := qbt.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error on login failure")
	}
}
