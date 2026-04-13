package transmission

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// transmissionFor builds a Transmission client pointed at the given httptest.Server.
func transmissionFor(t *testing.T, srv *httptest.Server) *Transmission {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return New(Settings{
		Host:    u.Hostname(),
		Port:    port,
		UrlBase: "/transmission/",
	}, srv.Client())
}

// rpcHandler creates a handler that simulates the 409-then-OK Transmission flow.
// It calls handlers[method] for the actual work on the second (authenticated) call.
func rpcServer(handlers map[string]func() interface{}) http.HandlerFunc {
	const fakeSessionID = "test-session-xyz"
	return func(w http.ResponseWriter, r *http.Request) {
		// Require the session ID header; if missing send 409.
		if r.Header.Get(sessionIDHeader) == "" {
			w.Header().Set(sessionIDHeader, fakeSessionID)
			w.WriteHeader(http.StatusConflict)
			return
		}

		var req struct {
			Method    string          `json:"method"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		h, ok := handlers[req.Method]
		if !ok {
			http.Error(w, "unknown method", http.StatusNotFound)
			return
		}

		result := h()
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"result":    "success",
			"arguments": result,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func TestTransmissionSessionIDRetry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Header.Get(sessionIDHeader) == "" {
			w.Header().Set(sessionIDHeader, "session-abc")
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","arguments":{}}`))
	}))
	defer srv.Close()

	tr := transmissionFor(t, srv)
	if err := tr.Test(context.Background()); err != nil {
		t.Fatalf("Test() returned error: %v", err)
	}
	// Expect 2 calls: one 409, one success.
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls for session handshake, got %d", callCount)
	}
}

func TestTransmissionAdd(t *testing.T) {
	srv := httptest.NewServer(rpcServer(map[string]func() interface{}{
		"torrent-add": func() interface{} {
			return map[string]interface{}{
				"torrent-added": map[string]interface{}{
					"id":   7,
					"name": "Show.S01E01",
				},
			}
		},
	}))
	defer srv.Close()

	tr := transmissionFor(t, srv)
	// Pre-seed session ID so we skip the 409 dance.
	tr.sessionID = "test-session-xyz"

	id, err := tr.Add(context.Background(), "https://example.com/test.torrent", "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "7" {
		t.Errorf("Add: got ID %q, want 7", id)
	}
}

func TestTransmissionItems(t *testing.T) {
	srv := httptest.NewServer(rpcServer(map[string]func() interface{}{
		"torrent-get": func() interface{} {
			return map[string]interface{}{
				"torrents": []map[string]interface{}{
					{
						"id":            3,
						"name":          "Show.S01E01",
						"status":        4,
						"totalSize":     1000000000,
						"leftUntilDone": 200000000,
						"downloadDir":   "/downloads/tv",
					},
				},
			}
		},
	}))
	defer srv.Close()

	tr := transmissionFor(t, srv)
	tr.sessionID = "test-session-xyz"

	items, err := tr.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.DownloadID != "3" {
		t.Errorf("DownloadID: got %q, want 3", item.DownloadID)
	}
	if item.Title != "Show.S01E01" {
		t.Errorf("Title: got %q", item.Title)
	}
	if item.Status != "downloading" {
		t.Errorf("Status: got %q, want downloading", item.Status)
	}
	if item.TotalSize != 1000000000 {
		t.Errorf("TotalSize: got %d", item.TotalSize)
	}
	if item.Remaining != 200000000 {
		t.Errorf("Remaining: got %d", item.Remaining)
	}
	if item.OutputPath != "/downloads/tv" {
		t.Errorf("OutputPath: got %q", item.OutputPath)
	}
}

func TestTransmissionTest(t *testing.T) {
	srv := httptest.NewServer(rpcServer(map[string]func() interface{}{
		"session-get": func() interface{} {
			return map[string]interface{}{"version": "4.0.5"}
		},
	}))
	defer srv.Close()

	tr := transmissionFor(t, srv)
	tr.sessionID = "test-session-xyz"
	if err := tr.Test(context.Background()); err != nil {
		t.Errorf("Test() returned error: %v", err)
	}
}
