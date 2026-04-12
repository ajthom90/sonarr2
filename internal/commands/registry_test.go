package commands

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	called := false
	r.Register("TestCommand", HandlerFunc(func(ctx context.Context, cmd Command) error {
		called = true
		return nil
	}))

	h, err := r.Get("TestCommand")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := h.Handle(context.Background(), Command{Name: "TestCommand"}); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	if !called {
		t.Error("expected handler to be called, but it was not")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("NoSuchCommand")
	if err == nil {
		t.Fatal("expected error for unregistered command, got nil")
	}
}

func TestHandlerFuncAdapter(t *testing.T) {
	var receivedCtx context.Context
	var receivedCmd Command

	sentinel := errors.New("sentinel error")

	fn := HandlerFunc(func(ctx context.Context, cmd Command) error {
		receivedCtx = ctx
		receivedCmd = cmd
		return sentinel
	})

	ctx := context.WithValue(context.Background(), ctxKey("key"), "value")
	cmd := Command{ID: 42, Name: "AdapterTest"}

	err := fn.Handle(ctx, cmd)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if receivedCtx != ctx {
		t.Error("context was not passed through to the underlying function")
	}
	if receivedCmd.ID != cmd.ID || receivedCmd.Name != cmd.Name {
		t.Errorf("command was not passed through: got %+v, want %+v", receivedCmd, cmd)
	}
}

// ctxKey is a private type to avoid collisions in context values.
type ctxKey string
