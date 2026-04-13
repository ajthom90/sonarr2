package realtime

import (
	"testing"
	"time"
)

// recv reads one message from c.ch with a short timeout to avoid test hangs.
func recv(t *testing.T, c *Client) (Message, bool) {
	t.Helper()
	select {
	case msg, ok := <-c.ch:
		return msg, ok
	case <-time.After(200 * time.Millisecond):
		return Message{}, false
	}
}

// noRecv asserts that no message arrives within a short window.
func noRecv(t *testing.T, c *Client) {
	t.Helper()
	select {
	case msg := <-c.ch:
		t.Errorf("unexpected message received: %+v", msg)
	case <-time.After(50 * time.Millisecond):
		// expected — nothing arrived
	}
}

func TestBrokerBroadcastToSubscribers(t *testing.T) {
	b := NewBroker(8)

	c1 := b.Subscribe()
	c2 := b.Subscribe()
	defer b.Unsubscribe(c1)
	defer b.Unsubscribe(c2)

	want := Message{Name: "series", Body: map[string]any{"action": "updated"}}
	b.Broadcast(want)

	msg1, ok1 := recv(t, c1)
	if !ok1 {
		t.Fatal("c1: expected message, channel closed or timeout")
	}
	if msg1.Name != want.Name {
		t.Errorf("c1 name: got %q, want %q", msg1.Name, want.Name)
	}

	msg2, ok2 := recv(t, c2)
	if !ok2 {
		t.Fatal("c2: expected message, channel closed or timeout")
	}
	if msg2.Name != want.Name {
		t.Errorf("c2 name: got %q, want %q", msg2.Name, want.Name)
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	b := NewBroker(8)
	c := b.Subscribe()

	b.Unsubscribe(c)

	// After unsubscribe the channel should be closed.
	select {
	case _, ok := <-c.ch:
		if ok {
			t.Error("expected channel to be closed, got a value instead")
		}
		// closed — correct
	case <-time.After(200 * time.Millisecond):
		t.Error("channel was not closed after Unsubscribe")
	}

	// Broadcast to an empty broker should not panic.
	b.Broadcast(Message{Name: "series", Body: nil})
}

func TestBrokerDropsSlowClient(t *testing.T) {
	bufSize := 2
	b := NewBroker(bufSize)
	slow := b.Subscribe()
	// Do not read from slow — its buffer will fill up.

	// Fill the slow client's buffer and then send one more.
	for i := 0; i <= bufSize+1; i++ {
		b.Broadcast(Message{Name: "series", Body: i})
	}

	// fast client should have received its messages without a hang.
	// The important assertion is that we reach this point at all
	// (Broadcast must not block). Drain slow to verify at most bufSize msgs.
	count := 0
	for {
		select {
		case _, ok := <-slow.ch:
			if !ok {
				goto done
			}
			count++
		default:
			goto done
		}
	}
done:
	if count > bufSize {
		t.Errorf("slow client received %d messages, expected at most %d (buffer size)", count, bufSize)
	}

	b.Unsubscribe(slow)
}

func TestBrokerFilter(t *testing.T) {
	b := NewBroker(8)

	// This client only wants "series" events.
	seriesOnly := b.Subscribe(MsgSeries)
	// This client wants everything.
	all := b.Subscribe()
	defer b.Unsubscribe(seriesOnly)
	defer b.Unsubscribe(all)

	// Broadcast a "command" message — seriesOnly must NOT receive it.
	b.Broadcast(Message{Name: string(MsgCommand), Body: nil})
	noRecv(t, seriesOnly)

	// "all" should have received the command.
	_, ok := recv(t, all)
	if !ok {
		t.Fatal("all: expected command message, got nothing")
	}

	// Broadcast a "series" message — seriesOnly MUST receive it.
	b.Broadcast(Message{Name: string(MsgSeries), Body: nil})
	_, ok = recv(t, seriesOnly)
	if !ok {
		t.Fatal("seriesOnly: expected series message, got nothing")
	}
}
