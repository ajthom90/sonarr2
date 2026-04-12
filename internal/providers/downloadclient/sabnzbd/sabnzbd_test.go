package sabnzbd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// canned JSON responses for testing.
const cannedAddResponse = `{"status": true, "nzo_ids": ["SABnzbd_nzo_abc123"]}`

const cannedQueueResponse = `{
  "queue": {
    "slots": [
      {
        "nzo_id": "SABnzbd_nzo_abc123",
        "filename": "Show.S01E01.720p",
        "status": "Downloading",
        "mb": "1500.00",
        "mbleft": "500.00",
        "storage": "/downloads/complete/tv/Show.S01E01.720p"
      }
    ]
  }
}`

const cannedVersionResponse = `{"version": "4.3.2"}`

// sabFor constructs a SABnzbd client pointed at the given httptest.Server.
func sabFor(t *testing.T, srv *httptest.Server) *SABnzbd {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL %q: %v", srv.URL, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("failed to parse test server port from %q: %v", srv.URL, err)
	}
	return New(Settings{
		Host:     u.Hostname(),
		Port:     port,
		ApiKey:   "testapikey",
		Category: "tv",
	}, srv.Client())
}

func TestSabnzbdAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("mode") != "addurl" {
			t.Errorf("expected mode=addurl, got %q", q.Get("mode"))
		}
		if q.Get("name") != "https://example.com/test.nzb" {
			t.Errorf("expected name=https://example.com/test.nzb, got %q", q.Get("name"))
		}
		if q.Get("nzbname") != "Show.S01E01" {
			t.Errorf("expected nzbname=Show.S01E01, got %q", q.Get("nzbname"))
		}
		if q.Get("cat") != "tv" {
			t.Errorf("expected cat=tv, got %q", q.Get("cat"))
		}
		if q.Get("apikey") != "testapikey" {
			t.Errorf("expected apikey=testapikey, got %q", q.Get("apikey"))
		}
		if q.Get("output") != "json" {
			t.Errorf("expected output=json, got %q", q.Get("output"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cannedAddResponse))
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	id, err := sab.Add(context.Background(), "https://example.com/test.nzb", "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "SABnzbd_nzo_abc123" {
		t.Errorf("expected download ID SABnzbd_nzo_abc123, got %q", id)
	}
}

func TestSabnzbdItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("mode") != "queue" {
			t.Errorf("expected mode=queue, got %q", q.Get("mode"))
		}
		if q.Get("apikey") != "testapikey" {
			t.Errorf("expected apikey=testapikey, got %q", q.Get("apikey"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cannedQueueResponse))
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	items, err := sab.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.DownloadID != "SABnzbd_nzo_abc123" {
		t.Errorf("DownloadID: got %q, want SABnzbd_nzo_abc123", item.DownloadID)
	}
	if item.Title != "Show.S01E01.720p" {
		t.Errorf("Title: got %q, want Show.S01E01.720p", item.Title)
	}
	if item.Status != "Downloading" {
		t.Errorf("Status: got %q, want Downloading", item.Status)
	}
	// 1500 MB in bytes = 1500 * 1024 * 1024 = 1572864000
	if item.TotalSize != 1572864000 {
		t.Errorf("TotalSize: got %d, want 1572864000", item.TotalSize)
	}
	// 500 MB in bytes = 500 * 1024 * 1024 = 524288000
	if item.Remaining != 524288000 {
		t.Errorf("Remaining: got %d, want 524288000", item.Remaining)
	}
	if item.OutputPath != "/downloads/complete/tv/Show.S01E01.720p" {
		t.Errorf("OutputPath: got %q, want /downloads/complete/tv/Show.S01E01.720p", item.OutputPath)
	}
}

func TestSabnzbdRemove(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": true}`))
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	if err := sab.Remove(context.Background(), "SABnzbd_nzo_abc123", true); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if gotQuery.Get("mode") != "queue" {
		t.Errorf("expected mode=queue, got %q", gotQuery.Get("mode"))
	}
	if gotQuery.Get("name") != "delete" {
		t.Errorf("expected name=delete, got %q", gotQuery.Get("name"))
	}
	if gotQuery.Get("value") != "SABnzbd_nzo_abc123" {
		t.Errorf("expected value=SABnzbd_nzo_abc123, got %q", gotQuery.Get("value"))
	}
	if gotQuery.Get("del_files") != "1" {
		t.Errorf("expected del_files=1, got %q", gotQuery.Get("del_files"))
	}
	if gotQuery.Get("apikey") != "testapikey" {
		t.Errorf("expected apikey=testapikey, got %q", gotQuery.Get("apikey"))
	}
}

func TestSabnzbdRemoveNoDelete(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": true}`))
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	if err := sab.Remove(context.Background(), "SABnzbd_nzo_abc123", false); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if gotQuery.Get("del_files") != "0" {
		t.Errorf("expected del_files=0, got %q", gotQuery.Get("del_files"))
	}
}

func TestSabnzbdTestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("mode") != "version" {
			t.Errorf("expected mode=version, got %q", q.Get("mode"))
		}
		if q.Get("apikey") != "testapikey" {
			t.Errorf("expected apikey=testapikey, got %q", q.Get("apikey"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cannedVersionResponse))
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	if err := sab.Test(context.Background()); err != nil {
		t.Errorf("Test() returned unexpected error: %v", err)
	}
}

func TestSabnzbdTestFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	sab := sabFor(t, srv)
	err := sab.Test(context.Background())
	if err == nil {
		t.Fatal("Test() should return an error for 401 response")
	}
}
