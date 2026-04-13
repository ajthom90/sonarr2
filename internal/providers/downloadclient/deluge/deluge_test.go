package deluge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// delugeFor builds a Deluge client pointed at the given httptest.Server.
func delugeFor(t *testing.T, srv *httptest.Server) *Deluge {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return New(Settings{
		Host:     u.Hostname(),
		Port:     port,
		Password: "testpass",
	}, srv.Client())
}

// rpcHandler routes JSON-RPC calls to per-method handlers.
// Each handler receives the raw params and returns the result value.
type methodHandler func(params json.RawMessage) interface{}

func delugeServer(handlers map[string]methodHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			ID     int             `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		h, ok := handlers[req.Method]
		if !ok {
			// Default: return true for unknown methods (e.g. auth.login).
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"result": true,
				"error":  nil,
				"id":     req.ID,
			})
			return
		}

		result := h(req.Params)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"result": result,
			"error":  nil,
			"id":     req.ID,
		})
	}
}

func TestDelugeAdd(t *testing.T) {
	srv := httptest.NewServer(delugeServer(map[string]methodHandler{
		"core.add_torrent_url": func(params json.RawMessage) interface{} {
			return "abcdef1234567890"
		},
	}))
	defer srv.Close()

	d := delugeFor(t, srv)
	id, err := d.Add(context.Background(), "https://example.com/test.torrent", "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "abcdef1234567890" {
		t.Errorf("Add: got ID %q, want abcdef1234567890", id)
	}
}

func TestDelugeItems(t *testing.T) {
	srv := httptest.NewServer(delugeServer(map[string]methodHandler{
		"core.get_torrents_status": func(params json.RawMessage) interface{} {
			return map[string]interface{}{
				"aabbcc": map[string]interface{}{
					"name":            "Show.S01E01",
					"state":           "Downloading",
					"total_size":      1000000000,
					"total_remaining": 300000000,
					"save_path":       "/downloads/tv",
				},
			}
		},
	}))
	defer srv.Close()

	d := delugeFor(t, srv)
	items, err := d.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.DownloadID != "aabbcc" {
		t.Errorf("DownloadID: got %q, want aabbcc", item.DownloadID)
	}
	if item.Title != "Show.S01E01" {
		t.Errorf("Title: got %q", item.Title)
	}
	if item.TotalSize != 1000000000 {
		t.Errorf("TotalSize: got %d", item.TotalSize)
	}
	if item.Remaining != 300000000 {
		t.Errorf("Remaining: got %d", item.Remaining)
	}
}

func TestDelugeTest(t *testing.T) {
	srv := httptest.NewServer(delugeServer(map[string]methodHandler{
		"daemon.info": func(params json.RawMessage) interface{} {
			return "2.1.1"
		},
	}))
	defer srv.Close()

	d := delugeFor(t, srv)
	if err := d.Test(context.Background()); err != nil {
		t.Errorf("Test() returned error: %v", err)
	}
}

func TestDelugeAuthFailure(t *testing.T) {
	srv := httptest.NewServer(delugeServer(map[string]methodHandler{
		"auth.login": func(params json.RawMessage) interface{} {
			return false // authentication failed
		},
	}))
	defer srv.Close()

	d := delugeFor(t, srv)
	if err := d.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error when auth.login returns false")
	}
}

func TestDelugeTestHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := delugeFor(t, srv)
	if err := d.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error for 500 response")
	}
}
