package events

import (
	"context"
	"reflect"
)

// noopBus is a Bus implementation that drops all events and silently accepts
// all subscriptions. Use in tests that don't care about event dispatch.
type noopBus struct{}

// NewNoopBus returns a Bus that drops all events.
func NewNoopBus() Bus { return noopBus{} }

func (noopBus) Publish(ctx context.Context, event any) error    { return nil }
func (noopBus) register(eventType reflect.Type, h handlerEntry) {}
