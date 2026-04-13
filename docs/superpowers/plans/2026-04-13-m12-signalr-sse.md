# Milestone 12 — SignalR Emulation + Server-Sent Events

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add real-time push so the UI and external clients get live updates when data changes (series added, episode grabbed, command progress, queue updates). Ship two transports: a SignalR-compatible WebSocket endpoint for existing Sonarr clients, and a cleaner SSE endpoint for new clients.

**Architecture:** `internal/realtime/` owns a `Broker` that subscribes to the internal event bus and fans out to connected clients. Two transport handlers mount on the HTTP server: `/signalr/messages` (WebSocket, Sonarr-compatible envelope) and `/api/v6/stream` (SSE).

---

## Layout

```
internal/
├── realtime/
│   ├── broker.go          # Broker: event bus → client channels
│   ├── broker_test.go
│   ├── signalr.go         # WebSocket handler emulating SignalR protocol
│   ├── signalr_test.go
│   ├── sse.go             # Server-Sent Events handler
│   ├── sse_test.go
│   └── message.go         # SignalR message envelope types
└── app/
    └── app.go             # Wire broker, start in Run
```

New dependency: `github.com/gorilla/websocket` for the SignalR WebSocket transport.

---

## Task 1 — Broker: event bus → client fanout

The broker subscribes to all domain events and broadcasts them to connected clients via per-client channels.

### broker.go

```go
package realtime

type MessageType string
const (
    MsgSeries      MessageType = "series"
    MsgEpisode     MessageType = "episode"
    MsgEpisodeFile MessageType = "episodefile"
    MsgCommand     MessageType = "command"
    MsgHealth      MessageType = "health"
    MsgQueue       MessageType = "queue"
    MsgCalendar    MessageType = "calendar"
)

type Message struct {
    Name string `json:"name"`
    Body any    `json:"body"`
}

type Client struct {
    ch     chan Message
    filter map[MessageType]bool // nil = all
}

type Broker struct {
    mu      sync.RWMutex
    clients map[*Client]struct{}
    bufSize int
}

func NewBroker(bufSize int) *Broker
func (b *Broker) Subscribe(filter ...MessageType) *Client
func (b *Broker) Unsubscribe(c *Client)
func (b *Broker) Broadcast(msg Message)
```

`Subscribe` creates a client with a buffered channel. `Broadcast` sends to all clients non-blocking (drops if buffer full — slow client loses messages rather than blocking the broker). `Unsubscribe` removes the client and closes its channel.

### Wiring to event bus

The broker subscribes async handlers for each domain event type and maps them to broadcast messages:

```go
func (b *Broker) SubscribeToEvents(bus events.Bus) {
    events.SubscribeAsync[library.SeriesAdded](bus, func(_ context.Context, e library.SeriesAdded) {
        b.Broadcast(Message{Name: "series", Body: map[string]any{"action": "updated", "resource": e}})
    })
    events.SubscribeAsync[library.SeriesDeleted](bus, func(_ context.Context, e library.SeriesDeleted) {
        b.Broadcast(Message{Name: "series", Body: map[string]any{"action": "deleted", "resource": e}})
    })
    events.SubscribeAsync[commands.CommandCompleted](bus, func(_ context.Context, e commands.CommandCompleted) {
        b.Broadcast(Message{Name: "command", Body: map[string]any{"action": "updated", "resource": e}})
    })
    // ... same for episode, episodefile, command started/failed
}
```

### Tests

- `TestBrokerBroadcastToSubscribers` — 2 clients subscribe, broadcast 1 message, both receive it
- `TestBrokerUnsubscribe` — unsubscribed client's channel is closed
- `TestBrokerDropsSlowClient` — fill a client's buffer, verify broadcast doesn't block
- `TestBrokerFilter` — client subscribes with filter for "series" only, verify it doesn't receive "command" messages

Commit: `feat(realtime): add event broker with per-client fanout and filtering`

---

## Task 2 — SignalR WebSocket transport

Emulate enough of the SignalR protocol for existing Sonarr clients (LunaSea, Notifiarr, etc.).

### SignalR wire protocol (simplified)

1. Client POSTs to `/signalr/messages/negotiate` → receives `{"connectionId":"...", "availableTransports":[{"transport":"WebSockets"}]}`
2. Client upgrades to WebSocket at `/signalr/messages?id={connectionId}`
3. Server sends messages as JSON: `{"H":"MessageHub","M":"receiveMessage","A":["series",{"action":"updated","resource":{...}}]}`

### signalr.go

```go
func (b *Broker) SignalRNegotiate(w http.ResponseWriter, r *http.Request)
func (b *Broker) SignalRConnect(w http.ResponseWriter, r *http.Request)
```

`Negotiate` returns a JSON response with a random connectionId and WebSockets as the available transport.

`Connect` upgrades to WebSocket via gorilla/websocket, subscribes to the broker, and writes each message in the SignalR envelope format. The goroutine reads from the client channel and writes to the WebSocket until the client disconnects or the context is cancelled.

### Tests

Use `httptest.NewServer` + `gorilla/websocket.Dial`:
- `TestSignalRNegotiateReturnsConnectionId` — POST negotiate, verify JSON
- `TestSignalRConnectReceivesMessages` — connect via WS, broadcast a message, verify the client receives it in SignalR envelope format
- `TestSignalRConnectDisconnectsCleanly` — close the WS, verify the client is unsubscribed

Commit: `feat(realtime): add SignalR WebSocket transport`

---

## Task 3 — Server-Sent Events transport

A cleaner alternative for new clients.

### sse.go

```go
func (b *Broker) SSEHandler(w http.ResponseWriter, r *http.Request)
```

Standard SSE: sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`. Subscribes to the broker, writes each message as:
```
event: series
data: {"action":"updated","resource":{...}}

```

Supports `?events=series,command` filter query param. Supports `Last-Event-ID` header for basic resume (not implemented in M12 — just acknowledge the header and don't break).

### Tests

Use `httptest.NewServer` + HTTP client with streaming read:
- `TestSSEReceivesMessages` — connect to SSE, broadcast, read event
- `TestSSEFiltersByEventType` — connect with `?events=series`, verify command events not received
- `TestSSEDisconnectsCleanly` — close connection, verify unsubscribed

Commit: `feat(realtime): add Server-Sent Events transport`

---

## Task 4 — Wire into app + mount routes + README + push

1. Add gorilla/websocket dependency: `go get github.com/gorilla/websocket@v1.5.3`
2. Create broker in app.New, subscribe to events
3. Mount routes:
   - `POST /signalr/messages/negotiate` (no auth — SignalR clients negotiate before auth in some impls; or gate behind auth — check what Sonarr does)
   - `GET /signalr/messages` (WebSocket upgrade)
   - `GET /api/v6/stream` (SSE, behind auth)
4. Start broker lifecycle (no background goroutines needed — broker is passive, handlers run in HTTP goroutines)
5. Update README: bump to M12, add real-time push to implemented list
6. tidy, lint, test, build, push, CI

---

## Done

After Task 4, the system pushes live updates to connected clients. External Sonarr clients (LunaSea, Notifiarr) connect via SignalR and receive series/episode/command updates in real time. New clients can use the cleaner SSE endpoint. M14 (frontend) will connect to one of these for live UI updates.
