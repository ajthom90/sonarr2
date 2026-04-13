package nzbget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// nzbgetFor builds a NZBGet client pointed at the given httptest.Server.
func nzbgetFor(t *testing.T, srv *httptest.Server) *NZBGet {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return New(Settings{
		Host:     u.Hostname(),
		Port:     port,
		Username: "testuser",
		Password: "testpass",
		Category: "tv",
	}, srv.Client())
}

// rpcHandler returns an http.HandlerFunc that reads the method from the JSON-RPC
// body and calls the appropriate handler function.
func rpcHandler(handlers map[string]func(w http.ResponseWriter, params json.RawMessage)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
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
		h(w, req.Params)
	}
}

func writeResult(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(map[string]interface{}{"result": result})
	_, _ = w.Write(data)
}

func TestNZBGetAdd(t *testing.T) {
	srv := httptest.NewServer(rpcHandler(map[string]func(http.ResponseWriter, json.RawMessage){
		"append": func(w http.ResponseWriter, params json.RawMessage) {
			writeResult(w, 42)
		},
	}))
	defer srv.Close()

	n := nzbgetFor(t, srv)
	id, err := n.Add(context.Background(), "https://example.com/test.nzb", "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "42" {
		t.Errorf("Add: got ID %q, want 42", id)
	}
}

func TestNZBGetItems(t *testing.T) {
	srv := httptest.NewServer(rpcHandler(map[string]func(http.ResponseWriter, json.RawMessage){
		"listgroups": func(w http.ResponseWriter, params json.RawMessage) {
			writeResult(w, []map[string]interface{}{
				{
					"NZBID":           1,
					"NZBFilename":     "Show.S01E01.720p.nzb",
					"Status":          "DOWNLOADING",
					"FileSizeMB":      1000,
					"RemainingSizeMB": 400,
					"DestDir":         "/downloads/tv",
				},
			})
		},
	}))
	defer srv.Close()

	n := nzbgetFor(t, srv)
	items, err := n.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.DownloadID != "1" {
		t.Errorf("DownloadID: got %q, want 1", item.DownloadID)
	}
	if item.Title != "Show.S01E01.720p.nzb" {
		t.Errorf("Title: got %q", item.Title)
	}
	if item.Status != "DOWNLOADING" {
		t.Errorf("Status: got %q", item.Status)
	}
	if item.TotalSize != 1000*1024*1024 {
		t.Errorf("TotalSize: got %d", item.TotalSize)
	}
	if item.Remaining != 400*1024*1024 {
		t.Errorf("Remaining: got %d", item.Remaining)
	}
	if item.OutputPath != "/downloads/tv" {
		t.Errorf("OutputPath: got %q", item.OutputPath)
	}
}

func TestNZBGetTest(t *testing.T) {
	srv := httptest.NewServer(rpcHandler(map[string]func(http.ResponseWriter, json.RawMessage){
		"version": func(w http.ResponseWriter, params json.RawMessage) {
			writeResult(w, "24.0")
		},
	}))
	defer srv.Close()

	n := nzbgetFor(t, srv)
	if err := n.Test(context.Background()); err != nil {
		t.Errorf("Test() returned error: %v", err)
	}
}

func TestNZBGetTestFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	n := nzbgetFor(t, srv)
	if err := n.Test(context.Background()); err == nil {
		t.Fatal("Test() should return an error for 401 response")
	}
}
