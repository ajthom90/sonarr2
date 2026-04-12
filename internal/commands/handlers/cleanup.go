package handlers

import (
	"context"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
)

// CleanupHandler deletes completed and failed commands older than 7 days.
// It is registered under the "MessagingCleanup" command name and run on a
// 1-hour schedule by the built-in task registered in app.New.
type CleanupHandler struct {
	queue commands.Queue
}

// NewCleanupHandler creates a CleanupHandler that operates on the given queue.
func NewCleanupHandler(queue commands.Queue) *CleanupHandler {
	return &CleanupHandler{queue: queue}
}

// Handle implements commands.Handler. It deletes commands whose ended_at is
// older than 7 days from now.
func (h *CleanupHandler) Handle(ctx context.Context, cmd commands.Command) error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	_, err := h.queue.DeleteOldCompleted(ctx, cutoff)
	return err
}
