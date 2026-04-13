// Package email implements a notification.Notification that sends email via SMTP.
package email

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Email sends notifications via SMTP.
type Email struct {
	settings Settings
}

// New constructs an Email notification provider.
func New(settings Settings) *Email {
	return &Email{settings: settings}
}

// Implementation satisfies providers.Provider.
func (e *Email) Implementation() string { return "Email" }

// DefaultName satisfies providers.Provider.
func (e *Email) DefaultName() string { return "Email" }

// Settings satisfies providers.Provider.
func (e *Email) Settings() any { return &e.settings }

// Test verifies the SMTP settings are valid.
func (e *Email) Test(_ context.Context) error {
	if e.settings.Server == "" {
		return fmt.Errorf("email: Server is not configured")
	}
	if e.settings.Port == 0 {
		return fmt.Errorf("email: Port is not configured")
	}
	if e.settings.From == "" {
		return fmt.Errorf("email: From is not configured")
	}
	if e.settings.To == "" {
		return fmt.Errorf("email: To is not configured")
	}
	return nil
}

// OnGrab sends a "Release Grabbed" email.
func (e *Email) OnGrab(_ context.Context, msg notification.GrabMessage) error {
	subject := "Release Grabbed: " + msg.SeriesTitle
	body := fmt.Sprintf("Release grabbed for %s — %s (%s) via %s", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality, msg.Indexer)
	return e.send(subject, body)
}

// OnDownload sends a "Download Complete" email.
func (e *Email) OnDownload(_ context.Context, msg notification.DownloadMessage) error {
	subject := "Download Complete: " + msg.SeriesTitle
	body := fmt.Sprintf("Download completed for %s — %s (%s)", msg.SeriesTitle, msg.EpisodeTitle, msg.Quality)
	return e.send(subject, body)
}

// OnHealthIssue sends a "Health Issue" email.
func (e *Email) OnHealthIssue(_ context.Context, msg notification.HealthMessage) error {
	subject := "Health Issue: " + msg.Type
	body := fmt.Sprintf("[%s] %s", msg.Type, msg.Message)
	return e.send(subject, body)
}

// send delivers an email using net/smtp.SendMail.
func (e *Email) send(subject, body string) error {
	addr := fmt.Sprintf("%s:%d", e.settings.Server, e.settings.Port)
	var auth smtp.Auth
	if e.settings.Username != "" {
		auth = smtp.PlainAuth("", e.settings.Username, e.settings.Password, e.settings.Server)
	}
	msg := []byte(
		"To: " + e.settings.To + "\r\n" +
			"From: " + e.settings.From + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body + "\r\n",
	)
	if err := smtp.SendMail(addr, auth, e.settings.From, []string{e.settings.To}, msg); err != nil {
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}
