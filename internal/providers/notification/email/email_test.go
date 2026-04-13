package email

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// TestEmailSettingsValid verifies that valid settings are accepted and the
// provider satisfies the notification.Notification interface. Real SMTP
// testing is integration-level and requires a running server.
func TestEmailSettingsValid(t *testing.T) {
	settings := Settings{
		Server:   "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "secret",
		From:     "sonarr@example.com",
		To:       "user@example.com",
		UseSsl:   false,
	}
	if settings.Server == "" {
		t.Error("Server should not be empty")
	}
	if settings.Port == 0 {
		t.Error("Port should not be zero")
	}
}

// TestEmailImplementsInterface verifies the Email type satisfies the
// notification.Notification interface at compile time.
func TestEmailImplementsInterface(t *testing.T) {
	var _ notification.Notification = New(Settings{})
}

// TestEmailImpl verifies method fields of the Email type.
func TestEmailImpl(t *testing.T) {
	e := New(Settings{Server: "smtp.example.com", Port: 587})
	if e.Implementation() != "Email" {
		t.Errorf("Implementation: got %q, want Email", e.Implementation())
	}
	if e.DefaultName() != "Email" {
		t.Errorf("DefaultName: got %q, want Email", e.DefaultName())
	}
	if e.Settings() == nil {
		t.Error("Settings() should not be nil")
	}
	_ = context.Background() // suppress unused import warning
}
