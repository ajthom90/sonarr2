package v3

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/commands"
)

// commandResource is the Sonarr v3 JSON shape for a command.
type commandResource struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	CommandName string          `json:"commandName"`
	Status      string          `json:"status"`
	Queued      string          `json:"queued"`
	Started     string          `json:"started"`
	Ended       string          `json:"ended"`
	Duration    string          `json:"duration"`
	Trigger     string          `json:"trigger"`
	Message     string          `json:"message"`
	Body        json.RawMessage `json:"body"`
	Priority    string          `json:"priority"`
}

// commandHandler handles /api/v3/command endpoints.
type commandHandler struct {
	queue commands.Queue
	log   *slog.Logger
}

// NewCommandHandler constructs a commandHandler.
func NewCommandHandler(queue commands.Queue, log *slog.Logger) *commandHandler {
	return &commandHandler{queue: queue, log: log}
}

// MountCommand registers /api/v3/command routes.
func MountCommand(r chi.Router, h *commandHandler) {
	r.Route("/api/v3/command", func(r chi.Router) {
		r.Post("/", h.enqueue)
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
	})
}

// toCommandResource converts a domain Command to its JSON shape.
func toCommandResource(c commands.Command) commandResource {
	started := ""
	if c.StartedAt != nil {
		started = formatTime(*c.StartedAt)
	}
	ended := ""
	if c.EndedAt != nil {
		ended = formatTime(*c.EndedAt)
	}
	duration := "00:00:00"
	if c.DurationMs != nil {
		d := time.Duration(*c.DurationMs) * time.Millisecond
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		s := int(d.Seconds()) % 60
		duration = fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}

	priorityName := "normal"
	switch c.Priority {
	case commands.PriorityHigh:
		priorityName = "high"
	case commands.PriorityLow:
		priorityName = "low"
	}

	body := c.Body
	if len(body) == 0 {
		body = json.RawMessage("{}")
	}

	// Produce a human-readable command name from the camelCase name.
	commandName := camelToWords(c.Name)

	return commandResource{
		ID:          c.ID,
		Name:        c.Name,
		CommandName: commandName,
		Status:      string(c.Status),
		Queued:      formatTime(c.QueuedAt),
		Started:     started,
		Ended:       ended,
		Duration:    duration,
		Trigger:     string(c.Trigger),
		Message:     c.Message,
		Body:        body,
		Priority:    priorityName,
	}
}

// camelToWords converts CamelCase to "Camel Case" words.
func camelToWords(s string) string {
	var out []byte
	for i, b := range []byte(s) {
		if i > 0 && b >= 'A' && b <= 'Z' {
			out = append(out, ' ')
		}
		out = append(out, b)
	}
	return string(out)
}

// commandEnqueueInput is the JSON body for POST /api/v3/command.
type commandEnqueueInput struct {
	Name string          `json:"name"`
	Body json.RawMessage `json:"body"`
}

// enqueue handles POST /api/v3/command.
func (h *commandHandler) enqueue(w http.ResponseWriter, r *http.Request) {
	var input commandEnqueueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	body := input.Body
	if len(body) == 0 {
		body = json.RawMessage("{}")
	}

	cmd, err := h.queue.Enqueue(r.Context(), input.Name, body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		h.log.Error("command enqueue", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toCommandResource(cmd))
}

// list handles GET /api/v3/command.
func (h *commandHandler) list(w http.ResponseWriter, r *http.Request) {
	cmds, err := h.queue.ListRecent(r.Context(), 50)
	if err != nil {
		h.log.Error("command list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if cmds == nil {
		cmds = []commands.Command{}
	}
	resources := make([]commandResource, 0, len(cmds))
	for _, c := range cmds {
		resources = append(resources, toCommandResource(c))
	}
	writeJSON(w, http.StatusOK, resources)
}

// get handles GET /api/v3/command/{id}.
func (h *commandHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	cmd, err := h.queue.Get(r.Context(), id)
	if errors.Is(err, commands.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("command get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toCommandResource(cmd))
}
