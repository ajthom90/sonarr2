package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPingHandler(t *testing.T) {
	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestStatusHandlerReturnsBuildInfo(t *testing.T) {
	origVersion := buildinfo.Version
	buildinfo.Version = "testversion"
	t.Cleanup(func() { buildinfo.Version = origVersion })

	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["appName"] != "sonarr2" {
		t.Errorf("appName = %v, want sonarr2", body["appName"])
	}
	if body["version"] != "testversion" {
		t.Errorf("version = %v, want testversion", body["version"])
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestStatusHandlerIncludesDatabase(t *testing.T) {
	ping := &stubPool{dialect: "sqlite", pingErr: nil}
	srv := httptest.NewServer(HandlerWithPool(discardLogger(), ping))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	dbv, ok := body["database"].(map[string]any)
	if !ok {
		t.Fatalf("database field missing or wrong type: %T", body["database"])
	}
	if dbv["dialect"] != "sqlite" {
		t.Errorf("dialect = %v", dbv["dialect"])
	}
	if dbv["connected"] != true {
		t.Errorf("connected = %v", dbv["connected"])
	}
}

func TestStatusHandlerReportsDatabaseDown(t *testing.T) {
	down := &stubPool{dialect: "postgres", pingErr: errPingDown}
	srv := httptest.NewServer(HandlerWithPool(discardLogger(), down))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	dbv, _ := body["database"].(map[string]any)
	if dbv["connected"] != false {
		t.Errorf("connected = %v, want false", dbv["connected"])
	}
}

// stubPool is a minimal test double for the pool shape used by the status
// handler. It is defined in the test file because it is only needed here.
type stubPool struct {
	dialect string
	pingErr error
}

func (s *stubPool) Dialect() string                { return s.dialect }
func (s *stubPool) Ping(ctx context.Context) error { return s.pingErr }

var errPingDown = stubPingError{}

type stubPingError struct{}

func (stubPingError) Error() string { return "db: ping down" }
