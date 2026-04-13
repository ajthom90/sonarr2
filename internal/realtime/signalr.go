package realtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Permissive origin check for M12 — tighten in a later milestone once
	// the allowed-origin list is known.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// randomID returns a 16-byte hex string suitable for use as a connection ID.
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is essentially fatal; panic is acceptable here.
		panic("realtime: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// SignalRNegotiate handles POST /signalr/messages/negotiate.
// It returns a NegotiateResponse that tells the client a random connection ID
// and that WebSockets is the only available transport.
func (b *Broker) SignalRNegotiate(w http.ResponseWriter, r *http.Request) {
	resp := NegotiateResponse{
		ConnectionID: randomID(),
		AvailableTransports: []TransportInfo{
			{
				Transport:       "WebSockets",
				TransferFormats: []string{"Text"},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("realtime: negotiate encode error: %v", err)
	}
}

// SignalRConnect handles GET /signalr/messages (WebSocket upgrade).
// It subscribes to the broker, then runs two concurrent loops:
//   - read loop: discards client messages; returns when the client disconnects.
//   - write loop: wraps each broker Message in a SignalRMessage and sends it.
//
// On disconnect (either side), the client is unsubscribed.
func (b *Broker) SignalRConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote an HTTP error response.
		log.Printf("realtime: ws upgrade error: %v", err)
		return
	}

	client := b.Subscribe()
	defer func() {
		b.Unsubscribe(client)
		conn.Close()
	}()

	// readDone is closed when the client disconnects so the write loop exits.
	readDone := make(chan struct{})

	// Read loop: discard incoming messages; detect disconnect.
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Write loop: forward broker messages to the WebSocket.
	const writeTimeout = 10 * time.Second
	for {
		select {
		case msg, ok := <-client.ch:
			if !ok {
				// Broker unsubscribed this client (e.g. broker shutdown).
				return
			}
			env := SignalRMessage{
				H: "MessageHub",
				M: "receiveMessage",
				A: []any{msg.Name, msg.Body},
			}
			conn.SetWriteDeadline(time.Now().Add(writeTimeout)) //nolint:errcheck
			if err := conn.WriteJSON(env); err != nil {
				log.Printf("realtime: ws write error: %v", err)
				return
			}
		case <-readDone:
			return
		case <-r.Context().Done():
			return
		}
	}
}
