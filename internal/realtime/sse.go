package realtime

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// SSEHandler handles GET /api/v6/stream.
//
// Supports an optional ?events=series,command query param; only the listed
// MessageTypes are delivered. With no ?events param, all messages are sent.
//
// Each message is written in standard SSE format:
//
//	event: {name}\n
//	data: {json body}\n
//	\n
//
// The Last-Event-ID header is accepted but not acted on in M12.
func (b *Broker) SSEHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse optional ?events= filter.
	var filter []MessageType
	if raw := r.URL.Query().Get("events"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				filter = append(filter, MessageType(part))
			}
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Disable buffering on nginx / proxies.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	client := b.Subscribe(filter...)
	defer b.Unsubscribe(client)

	for {
		select {
		case msg, ok := <-client.ch:
			if !ok {
				// Broker closed the channel (e.g. broker shutdown).
				return
			}
			body, err := json.Marshal(msg.Body)
			if err != nil {
				log.Printf("realtime: sse marshal error: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Name, body)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
