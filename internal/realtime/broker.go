// Package realtime provides an event broker that subscribes to the internal
// domain event bus and fans out messages to connected clients via per-client
// buffered channels. Two transport handlers (SignalR WebSocket and SSE) sit on
// top of the broker and are mounted by the HTTP server.
package realtime

import (
	"context"
	"sync"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
)

// MessageType identifies the domain entity that changed.
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

// Message is the unit passed through client channels and across the wire.
type Message struct {
	Name string `json:"name"`
	Body any    `json:"body"`
}

// Client represents a single connected consumer. The ch field is the
// per-client delivery channel; filter, when non-nil, limits which
// MessageTypes are delivered to this client.
type Client struct {
	ch     chan Message
	filter map[MessageType]bool // nil means accept all
}

// Broker fans out domain events to all connected clients.
type Broker struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	bufSize int
}

// NewBroker creates a Broker whose per-client channels have the given buffer
// size. A bufSize of 0 creates unbuffered channels (not recommended in
// production — slow clients will block Broadcast).
func NewBroker(bufSize int) *Broker {
	return &Broker{
		clients: make(map[*Client]struct{}),
		bufSize: bufSize,
	}
}

// Subscribe creates and registers a new Client. If one or more MessageType
// values are provided, the client only receives messages whose Name matches
// one of them. With no filter arguments the client receives all messages.
func (b *Broker) Subscribe(filter ...MessageType) *Client {
	c := &Client{
		ch: make(chan Message, b.bufSize),
	}
	if len(filter) > 0 {
		c.filter = make(map[MessageType]bool, len(filter))
		for _, f := range filter {
			c.filter[f] = true
		}
	}

	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()

	return c
}

// Unsubscribe removes the client from the broker and closes its channel so
// transport handlers can detect the disconnect via a range loop.
func (b *Broker) Unsubscribe(c *Client) {
	b.mu.Lock()
	delete(b.clients, c)
	b.mu.Unlock()

	close(c.ch)
}

// Broadcast delivers msg to every registered client whose filter allows it.
// Delivery is non-blocking: if a client's buffer is full the message is
// silently dropped for that client rather than blocking the caller.
func (b *Broker) Broadcast(msg Message) {
	b.mu.RLock()
	snapshot := make([]*Client, 0, len(b.clients))
	for c := range b.clients {
		snapshot = append(snapshot, c)
	}
	b.mu.RUnlock()

	for _, c := range snapshot {
		if c.filter != nil && !c.filter[MessageType(msg.Name)] {
			continue
		}
		select {
		case c.ch <- msg:
		default:
			// slow client — drop rather than block
		}
	}
}

// SubscribeToEvents registers async event bus handlers so that every relevant
// domain event triggers a Broadcast. Async handlers are used so that real-time
// push cannot block the domain operation that published the event.
func (b *Broker) SubscribeToEvents(bus events.Bus) {
	events.SubscribeAsync[library.SeriesAdded](bus, func(_ context.Context, e library.SeriesAdded) {
		b.Broadcast(Message{Name: string(MsgSeries), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[library.SeriesUpdated](bus, func(_ context.Context, e library.SeriesUpdated) {
		b.Broadcast(Message{Name: string(MsgSeries), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[library.SeriesDeleted](bus, func(_ context.Context, e library.SeriesDeleted) {
		b.Broadcast(Message{Name: string(MsgSeries), Body: map[string]any{"action": "deleted", "resource": e}})
	})

	events.SubscribeAsync[library.EpisodeAdded](bus, func(_ context.Context, e library.EpisodeAdded) {
		b.Broadcast(Message{Name: string(MsgEpisode), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[library.EpisodeUpdated](bus, func(_ context.Context, e library.EpisodeUpdated) {
		b.Broadcast(Message{Name: string(MsgEpisode), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[library.EpisodeDeleted](bus, func(_ context.Context, e library.EpisodeDeleted) {
		b.Broadcast(Message{Name: string(MsgEpisode), Body: map[string]any{"action": "deleted", "resource": e}})
	})

	events.SubscribeAsync[library.EpisodeFileAdded](bus, func(_ context.Context, e library.EpisodeFileAdded) {
		b.Broadcast(Message{Name: string(MsgEpisodeFile), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[library.EpisodeFileDeleted](bus, func(_ context.Context, e library.EpisodeFileDeleted) {
		b.Broadcast(Message{Name: string(MsgEpisodeFile), Body: map[string]any{"action": "deleted", "resource": e}})
	})

	events.SubscribeAsync[commands.CommandStarted](bus, func(_ context.Context, e commands.CommandStarted) {
		b.Broadcast(Message{Name: string(MsgCommand), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[commands.CommandCompleted](bus, func(_ context.Context, e commands.CommandCompleted) {
		b.Broadcast(Message{Name: string(MsgCommand), Body: map[string]any{"action": "updated", "resource": e}})
	})
	events.SubscribeAsync[commands.CommandFailed](bus, func(_ context.Context, e commands.CommandFailed) {
		b.Broadcast(Message{Name: string(MsgCommand), Body: map[string]any{"action": "failed", "resource": e}})
	})
}
