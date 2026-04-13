package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/providers"
	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// notificationResource is the Sonarr v3 JSON shape for a notification instance.
type notificationResource struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	Settings       json.RawMessage `json:"fields"`
	OnGrab         bool            `json:"onGrab"`
	OnDownload     bool            `json:"onDownload"`
	OnHealthIssue  bool            `json:"onHealthIssue"`
	Tags           json.RawMessage `json:"tags"`
	Added          string          `json:"added"`
}

// notificationSchemaResource is the schema for a notification implementation.
type notificationSchemaResource struct {
	Implementation string          `json:"implementation"`
	Name           string          `json:"name"`
	Fields         json.RawMessage `json:"fields"`
}

// notificationHandler handles /api/v3/notification endpoints.
type notificationHandler struct {
	store    notification.InstanceStore
	registry *notification.Registry
	log      *slog.Logger
}

// NewNotificationHandler constructs a notificationHandler.
func NewNotificationHandler(store notification.InstanceStore, registry *notification.Registry, log *slog.Logger) *notificationHandler {
	return &notificationHandler{store: store, registry: registry, log: log}
}

// MountNotification registers /api/v3/notification routes.
func MountNotification(r chi.Router, h *notificationHandler) {
	r.Route("/api/v3/notification", func(r chi.Router) {
		r.Get("/schema", h.schema)
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toNotificationResource(inst notification.Instance) notificationResource {
	settings := inst.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("[]")
	}
	tags := inst.Tags
	if len(tags) == 0 {
		tags = json.RawMessage("[]")
	}
	return notificationResource{
		ID:             inst.ID,
		Name:           inst.Name,
		Implementation: inst.Implementation,
		Settings:       settings,
		OnGrab:         inst.OnGrab,
		OnDownload:     inst.OnDownload,
		OnHealthIssue:  inst.OnHealthIssue,
		Tags:           tags,
		Added:          formatTime(inst.Added),
	}
}

// schema handles GET /api/v3/notification/schema.
func (h *notificationHandler) schema(w http.ResponseWriter, r *http.Request) {
	all := h.registry.All()
	result := make([]notificationSchemaResource, 0, len(all))
	for name, factory := range all {
		impl := factory()
		schema := providers.SchemaFor(impl.Settings())
		fieldsJSON, _ := json.Marshal(schema.Fields)
		result = append(result, notificationSchemaResource{
			Implementation: name,
			Name:           impl.DefaultName(),
			Fields:         json.RawMessage(fieldsJSON),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// list handles GET /api/v3/notification.
func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("notification list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]notificationResource, 0, len(all))
	for _, inst := range all {
		resources = append(resources, toNotificationResource(inst))
	}
	writeJSON(w, http.StatusOK, resources)
}

// notificationInput is the JSON body for POST/PUT notification.
type notificationInput struct {
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	Settings       json.RawMessage `json:"fields"`
	OnGrab         bool            `json:"onGrab"`
	OnDownload     bool            `json:"onDownload"`
	OnHealthIssue  bool            `json:"onHealthIssue"`
	Tags           json.RawMessage `json:"tags"`
}

// create handles POST /api/v3/notification.
func (h *notificationHandler) create(w http.ResponseWriter, r *http.Request) {
	var input notificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	settings := input.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	inst, err := h.store.Create(r.Context(), notification.Instance{
		Name:           input.Name,
		Implementation: input.Implementation,
		Settings:       settings,
		OnGrab:         input.OnGrab,
		OnDownload:     input.OnDownload,
		OnHealthIssue:  input.OnHealthIssue,
		Tags:           input.Tags,
	})
	if err != nil {
		h.log.Error("notification create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toNotificationResource(inst))
}

// get handles GET /api/v3/notification/{id}.
func (h *notificationHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	inst, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, notification.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("notification get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toNotificationResource(inst))
}

// update handles PUT /api/v3/notification/{id}.
func (h *notificationHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	existing, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, notification.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("notification update get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input notificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if len(input.Settings) > 0 {
		existing.Settings = input.Settings
	}
	existing.OnGrab = input.OnGrab
	existing.OnDownload = input.OnDownload
	existing.OnHealthIssue = input.OnHealthIssue
	if len(input.Tags) > 0 {
		existing.Tags = input.Tags
	}

	if err := h.store.Update(r.Context(), existing); err != nil {
		h.log.Error("notification update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toNotificationResource(existing))
}

// delete handles DELETE /api/v3/notification/{id}.
func (h *notificationHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, notification.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("notification delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
