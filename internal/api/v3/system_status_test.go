package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// stubStatusPool is a test double for PoolPinger used by system/status tests.
type stubStatusPool struct {
	dialect string
	pingErr error
}

func (s *stubStatusPool) Dialect() string              { return s.dialect }
func (s *stubStatusPool) Ping(_ context.Context) error { return s.pingErr }

func TestSystemStatusFullFields(t *testing.T) {
	r := chi.NewRouter()
	h := NewSystemStatusHandler(&stubStatusPool{dialect: "sqlite"})
	MountSystemStatus(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/system/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", ct)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Fields that Sonarr clients actually read.
	requiredFields := []string{
		"appName",
		"instanceName",
		"version",
		"buildTime",
		"isDebug",
		"isProduction",
		"isLinux",
		"isOsx",
		"isWindows",
		"isDocker",
		"branch",
		"databaseType",
		"databaseVersion",
		"authentication",
		"startTime",
		"urlBase",
		"runtimeName",
		"runtimeVersion",
		"startupPath",
		"appData",
		"mode",
		"instanceName",
		"packageVersion",
		"packageAuthor",
		"packageUpdateMechanism",
	}
	for _, field := range requiredFields {
		if _, ok := body[field]; !ok {
			t.Errorf("missing required field %q", field)
		}
	}

	// Spot-check specific values.
	if body["appName"] != "sonarr2" {
		t.Errorf("appName = %v, want sonarr2", body["appName"])
	}
	if body["instanceName"] != "sonarr2" {
		t.Errorf("instanceName = %v, want sonarr2", body["instanceName"])
	}
	if body["runtimeName"] != "go" {
		t.Errorf("runtimeName = %v, want go", body["runtimeName"])
	}
	if body["mode"] != "console" {
		t.Errorf("mode = %v, want console", body["mode"])
	}
	if body["authentication"] != "forms" {
		t.Errorf("authentication = %v, want forms", body["authentication"])
	}
	if body["isDebug"] != false {
		t.Errorf("isDebug = %v, want false", body["isDebug"])
	}
	if body["isProduction"] != true {
		t.Errorf("isProduction = %v, want true", body["isProduction"])
	}
	if body["packageUpdateMechanism"] != "docker" {
		t.Errorf("packageUpdateMechanism = %v, want docker", body["packageUpdateMechanism"])
	}
	// runtimeVersion should be non-empty and not contain "go" prefix.
	rv, _ := body["runtimeVersion"].(string)
	if rv == "" {
		t.Error("runtimeVersion should be non-empty")
	}
	if len(rv) >= 2 && rv[:2] == "go" {
		t.Errorf("runtimeVersion %q should not have 'go' prefix", rv)
	}
}

func TestSystemStatusNilPool(t *testing.T) {
	r := chi.NewRouter()
	h := NewSystemStatusHandler(nil)
	MountSystemStatus(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/system/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even with nil pool", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["databaseType"] != "sqLite" {
		t.Errorf("databaseType = %v, want sqLite for nil pool", body["databaseType"])
	}
}

func TestSystemStatusPostgresPool(t *testing.T) {
	r := chi.NewRouter()
	h := NewSystemStatusHandler(&stubStatusPool{dialect: "postgres"})
	MountSystemStatus(r, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/system/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["databaseType"] != "postgresql" {
		t.Errorf("databaseType = %v, want postgresql for postgres pool", body["databaseType"])
	}
}
