package tvdb_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/tvdb"
)

// TestRateLimitedTransport_PassesThrough verifies that requests succeed and the
// inner transport is actually called.
func TestRateLimitedTransport_PassesThrough(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	inner := ts.Client().Transport
	transport := tvdb.NewRateLimitedTransport(inner, tvdb.RateLimitOptions{
		RequestsPerSecond: 100,
		Burst:             10,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("inner transport called %d times, want 1", n)
	}
}

// TestRateLimitedTransport_Retries429 verifies that a 429 response is retried
// and the final success response is returned.
func TestRateLimitedTransport_Retries429(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			// First two attempts return 429; use Retry-After: 0 to keep test fast.
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	inner := ts.Client().Transport
	transport := tvdb.NewRateLimitedTransport(inner, tvdb.RateLimitOptions{
		RequestsPerSecond: 100,
		Burst:             10,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/data", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("inner transport called %d times, want 3", n)
	}
}

// TestRateLimitedTransport_MaxRetriesExceeded verifies that after exhausting all
// retries the final 429 response is returned (not an error).
func TestRateLimitedTransport_MaxRetriesExceeded(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	inner := ts.Client().Transport
	transport := tvdb.NewRateLimitedTransport(inner, tvdb.RateLimitOptions{
		RequestsPerSecond: 100,
		Burst:             10,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/data", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
	// 1 initial attempt + 3 retries = 4 total calls.
	if n := calls.Load(); n != 4 {
		t.Errorf("inner transport called %d times, want 4", n)
	}
}

// TestRateLimitedTransport_RespectsRetryAfterHeader verifies that when a 429
// carries a Retry-After header the transport waits at least that duration.
func TestRateLimitedTransport_RespectsRetryAfterHeader(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// First attempt: signal a 1-second back-off.
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	inner := ts.Client().Transport
	transport := tvdb.NewRateLimitedTransport(inner, tvdb.RateLimitOptions{
		RequestsPerSecond: 100,
		Burst:             10,
		MaxRetries:        3,
	})
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/slow", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v, want >= 900ms (Retry-After: 1 should introduce ~1s delay)", elapsed)
	}
}
