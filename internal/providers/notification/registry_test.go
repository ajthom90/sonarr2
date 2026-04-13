package notification

import (
	"context"
	"strings"
	"testing"
)

// stubNotification is a minimal implementation of Notification used only in tests.
type stubNotification struct{}

func (s *stubNotification) Implementation() string                        { return "Stub" }
func (s *stubNotification) DefaultName() string                           { return "Stub Notification" }
func (s *stubNotification) Settings() any                                 { return nil }
func (s *stubNotification) Test(_ context.Context) error                  { return nil }
func (s *stubNotification) OnGrab(_ context.Context, _ GrabMessage) error { return nil }
func (s *stubNotification) OnDownload(_ context.Context, _ DownloadMessage) error {
	return nil
}
func (s *stubNotification) OnHealthIssue(_ context.Context, _ HealthMessage) error {
	return nil
}

func TestNotifRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register("Stub", func() Notification { return &stubNotification{} })

	f, err := r.Get("Stub")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	n := f()
	if n.Implementation() != "Stub" {
		t.Errorf("Implementation() = %q, want %q", n.Implementation(), "Stub")
	}
}

func TestNotifRegistryGetMissing(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("DoesNotExist")
	if err == nil {
		t.Fatal("expected error for missing factory, got nil")
	}
	if !strings.Contains(err.Error(), "DoesNotExist") {
		t.Errorf("error message %q does not mention the missing key", err.Error())
	}
}

func TestNotifRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register("A", func() Notification { return &stubNotification{} })
	r.Register("B", func() Notification { return &stubNotification{} })

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d entries, want 2", len(all))
	}
	for _, name := range []string{"A", "B"} {
		if _, ok := all[name]; !ok {
			t.Errorf("All() missing key %q", name)
		}
	}
}

func TestNotifRegistryDuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register("Dup", func() Notification { return &stubNotification{} })

	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic for duplicate registration, got none")
		}
	}()
	r.Register("Dup", func() Notification { return &stubNotification{} })
}
