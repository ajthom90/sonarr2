package realtime

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// readSSELines reads SSE lines from the given scanner until it finds a non-empty
// line that starts with the given prefix, or the deadline is exceeded.
func readSSELine(t *testing.T, scanner *bufio.Scanner, prefix string) string {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, prefix) {
				done <- line
				return
			}
		}
		done <- ""
	}()
	select {
	case line := <-done:
		return line
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timeout waiting for SSE line with prefix %q", prefix)
		return ""
	}
}

func TestSSEReceivesMessages(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", b.SSEHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Connect to the SSE endpoint with a context we can cancel later.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want %q", ct, "text/event-stream")
	}

	// Give the handler time to subscribe.
	time.Sleep(20 * time.Millisecond)

	b.Broadcast(Message{Name: "series", Body: map[string]any{"action": "updated"}})

	scanner := bufio.NewScanner(resp.Body)
	eventLine := readSSELine(t, scanner, "event:")
	if eventLine == "" {
		t.Fatal("no event line received")
	}
	if !strings.Contains(eventLine, "series") {
		t.Errorf("event line %q does not contain 'series'", eventLine)
	}
	dataLine := readSSELine(t, scanner, "data:")
	if dataLine == "" {
		t.Fatal("no data line received")
	}
	if !strings.Contains(dataLine, "updated") {
		t.Errorf("data line %q does not contain 'updated'", dataLine)
	}
}

func TestSSEFiltersByEventType(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", b.SSEHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream?events=series", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /stream?events=series: %v", err)
	}
	defer resp.Body.Close()

	// Give the handler time to subscribe.
	time.Sleep(20 * time.Millisecond)

	// Broadcast a "command" event — should NOT arrive.
	b.Broadcast(Message{Name: "command", Body: map[string]any{"action": "updated"}})

	// Broadcast a "series" event — SHOULD arrive.
	b.Broadcast(Message{Name: "series", Body: map[string]any{"action": "updated", "id": 42}})

	scanner := bufio.NewScanner(resp.Body)
	eventLine := readSSELine(t, scanner, "event:")
	if eventLine == "" {
		t.Fatal("no event line received for series event")
	}
	if !strings.Contains(eventLine, "series") {
		t.Errorf("expected 'series' event but got: %q", eventLine)
	}
	if strings.Contains(eventLine, "command") {
		t.Errorf("received 'command' event even though filter was 'series': %q", eventLine)
	}
}

func TestSSEDisconnectsCleanly(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", b.SSEHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()

	// Give the handler time to subscribe.
	time.Sleep(20 * time.Millisecond)

	b.mu.RLock()
	nBefore := len(b.clients)
	b.mu.RUnlock()
	if nBefore == 0 {
		t.Fatal("expected at least one client before cancel")
	}

	// Cancel the client context — simulates client disconnect.
	cancel()

	// Drain the response body so the server detects EOF.
	go func() { //nolint:errcheck
		buf := make([]byte, 64)
		for {
			_, err := resp.Body.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	// Wait for the server-side handler to unsubscribe.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		b.mu.RLock()
		n := len(b.clients)
		b.mu.RUnlock()
		if n == 0 {
			return // clean disconnect confirmed
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("SSE client was not unsubscribed after context cancel")
}
