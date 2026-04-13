package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/providers"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

// downloadClientResource is the Sonarr v3 JSON shape for a download client.
type downloadClientResource struct {
	ID                       int             `json:"id"`
	Name                     string          `json:"name"`
	Implementation           string          `json:"implementation"`
	Settings                 json.RawMessage `json:"fields"`
	Enable                   bool            `json:"enable"`
	Priority                 int             `json:"priority"`
	RemoveCompletedDownloads bool            `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool            `json:"removeFailedDownloads"`
	Added                    string          `json:"added"`
}

// downloadClientSchemaResource is the schema for a download client implementation.
type downloadClientSchemaResource struct {
	Implementation string          `json:"implementation"`
	Name           string          `json:"name"`
	Fields         json.RawMessage `json:"fields"`
}

// downloadClientHandler handles /api/v3/downloadclient endpoints.
type downloadClientHandler struct {
	store    downloadclient.InstanceStore
	registry *downloadclient.Registry
	log      *slog.Logger
}

// NewDownloadClientHandler constructs a downloadClientHandler.
func NewDownloadClientHandler(store downloadclient.InstanceStore, registry *downloadclient.Registry, log *slog.Logger) *downloadClientHandler {
	return &downloadClientHandler{store: store, registry: registry, log: log}
}

// MountDownloadClient registers /api/v3/downloadclient routes.
func MountDownloadClient(r chi.Router, h *downloadClientHandler) {
	r.Route("/api/v3/downloadclient", func(r chi.Router) {
		r.Get("/schema", h.schema)
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toDownloadClientResource(inst downloadclient.Instance) downloadClientResource {
	settings := inst.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("[]")
	}
	return downloadClientResource{
		ID:                       inst.ID,
		Name:                     inst.Name,
		Implementation:           inst.Implementation,
		Settings:                 settings,
		Enable:                   inst.Enable,
		Priority:                 inst.Priority,
		RemoveCompletedDownloads: inst.RemoveCompletedDownloads,
		RemoveFailedDownloads:    inst.RemoveFailedDownloads,
		Added:                    formatTime(inst.Added),
	}
}

// schema handles GET /api/v3/downloadclient/schema.
func (h *downloadClientHandler) schema(w http.ResponseWriter, r *http.Request) {
	all := h.registry.All()
	result := make([]downloadClientSchemaResource, 0, len(all))
	for name, factory := range all {
		impl := factory()
		schema := providers.SchemaFor(impl.Settings())
		fieldsJSON, _ := json.Marshal(schema.Fields)
		result = append(result, downloadClientSchemaResource{
			Implementation: name,
			Name:           impl.DefaultName(),
			Fields:         json.RawMessage(fieldsJSON),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// list handles GET /api/v3/downloadclient.
func (h *downloadClientHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("downloadclient list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]downloadClientResource, 0, len(all))
	for _, inst := range all {
		resources = append(resources, toDownloadClientResource(inst))
	}
	writeJSON(w, http.StatusOK, resources)
}

// downloadClientInput is the JSON body for POST/PUT download client.
type downloadClientInput struct {
	Name                     string          `json:"name"`
	Implementation           string          `json:"implementation"`
	Settings                 json.RawMessage `json:"fields"`
	Enable                   bool            `json:"enable"`
	Priority                 int             `json:"priority"`
	RemoveCompletedDownloads bool            `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool            `json:"removeFailedDownloads"`
}

// create handles POST /api/v3/downloadclient.
func (h *downloadClientHandler) create(w http.ResponseWriter, r *http.Request) {
	var input downloadClientInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	settings := input.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	inst, err := h.store.Create(r.Context(), downloadclient.Instance{
		Name:                     input.Name,
		Implementation:           input.Implementation,
		Settings:                 settings,
		Enable:                   input.Enable,
		Priority:                 input.Priority,
		RemoveCompletedDownloads: input.RemoveCompletedDownloads,
		RemoveFailedDownloads:    input.RemoveFailedDownloads,
	})
	if err != nil {
		h.log.Error("downloadclient create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toDownloadClientResource(inst))
}

// get handles GET /api/v3/downloadclient/{id}.
func (h *downloadClientHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	inst, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, downloadclient.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("downloadclient get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toDownloadClientResource(inst))
}

// update handles PUT /api/v3/downloadclient/{id}.
func (h *downloadClientHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	existing, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, downloadclient.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("downloadclient update get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input downloadClientInput
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
	existing.Enable = input.Enable
	existing.RemoveCompletedDownloads = input.RemoveCompletedDownloads
	existing.RemoveFailedDownloads = input.RemoveFailedDownloads
	if input.Priority > 0 {
		existing.Priority = input.Priority
	}

	if err := h.store.Update(r.Context(), existing); err != nil {
		h.log.Error("downloadclient update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toDownloadClientResource(existing))
}

// delete handles DELETE /api/v3/downloadclient/{id}.
func (h *downloadClientHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, downloadclient.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("downloadclient delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
