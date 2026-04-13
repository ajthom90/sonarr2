package realtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialWS is a helper that connects to the given httptest.Server URL path over
// WebSocket (ws://) and returns the connection.
func dialWS(t *testing.T, srv *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	u := "ws" + strings.TrimPrefix(srv.URL, "http") + path
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("ws dial %s: %v", u, err)
	}
	return conn
}

func TestSignalRNegotiateReturnsConnectionId(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/signalr/messages/negotiate", b.SignalRNegotiate)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/signalr/messages/negotiate", "application/json", nil)
	if err != nil {
		t.Fatalf("POST negotiate: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}

	var neg NegotiateResponse
	if err := json.NewDecoder(resp.Body).Decode(&neg); err != nil {
		t.Fatalf("decode negotiate response: %v", err)
	}
	if neg.ConnectionID == "" {
		t.Error("connectionId is empty")
	}
	if len(neg.AvailableTransports) == 0 {
		t.Error("availableTransports is empty")
	}
	found := false
	for _, tr := range neg.AvailableTransports {
		if tr.Transport == "WebSockets" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("WebSockets not in availableTransports: %+v", neg.AvailableTransports)
	}

	// Two negotiate calls must return distinct IDs.
	resp2, err := http.Post(srv.URL+"/signalr/messages/negotiate", "application/json", nil)
	if err != nil {
		t.Fatalf("second POST: %v", err)
	}
	defer resp2.Body.Close()
	var neg2 NegotiateResponse
	if err := json.NewDecoder(resp2.Body).Decode(&neg2); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if neg.ConnectionID == neg2.ConnectionID {
		t.Error("two negotiate calls returned the same connectionId")
	}
}

func TestSignalRConnectReceivesMessages(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/signalr/messages", b.SignalRConnect)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv, "/signalr/messages")
	defer conn.Close()

	// Give the handler goroutine a moment to register the client.
	time.Sleep(20 * time.Millisecond)

	b.Broadcast(Message{Name: "series", Body: map[string]any{"action": "updated"}})

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) //nolint:errcheck
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}

	var env SignalRMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal SignalRMessage: %v", err)
	}
	if env.H != "MessageHub" {
		t.Errorf("H: got %q, want %q", env.H, "MessageHub")
	}
	if env.M != "receiveMessage" {
		t.Errorf("M: got %q, want %q", env.M, "receiveMessage")
	}
	if len(env.A) != 2 {
		t.Fatalf("A: got len=%d, want 2; A=%v", len(env.A), env.A)
	}
	if name, _ := env.A[0].(string); name != "series" {
		t.Errorf("A[0]: got %q, want %q", name, "series")
	}
}

func TestSignalRConnectDisconnectsCleanly(t *testing.T) {
	b := NewBroker(8)
	mux := http.NewServeMux()
	mux.HandleFunc("/signalr/messages", b.SignalRConnect)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv, "/signalr/messages")

	// Give the handler goroutine a moment to subscribe.
	time.Sleep(20 * time.Millisecond)

	countBefore := func() int {
		b.mu.RLock()
		defer b.mu.RUnlock()
		return len(b.clients)
	}

	if n := countBefore(); n == 0 {
		t.Fatal("expected at least one client before disconnect")
	}

	// Close from the client side.
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")) //nolint:errcheck
	conn.Close()

	// Allow the server-side handler to detect the close and unsubscribe.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		b.mu.RLock()
		n := len(b.clients)
		b.mu.RUnlock()
		if n == 0 {
			return // clean
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("client was not unsubscribed after WebSocket close")
}
