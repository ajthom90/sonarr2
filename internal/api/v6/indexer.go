package v6

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/providers"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// indexerResource is the v6 JSON shape for an indexer instance.
type indexerResource struct {
	ID                      int             `json:"id"`
	Name                    string          `json:"name"`
	Implementation          string          `json:"implementation"`
	Settings                json.RawMessage `json:"fields"`
	EnableRss               bool            `json:"enableRss"`
	EnableAutomaticSearch   bool            `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool            `json:"enableInteractiveSearch"`
	Priority                int             `json:"priority"`
	Added                   string          `json:"added"`
}

// indexerSchemaResource is the schema for an indexer implementation.
type indexerSchemaResource struct {
	Implementation string          `json:"implementation"`
	Name           string          `json:"name"`
	Fields         json.RawMessage `json:"fields"`
}

type indexerHandler struct {
	store    indexer.InstanceStore
	registry *indexer.Registry
	log      *slog.Logger
}

func newIndexerHandler(store indexer.InstanceStore, registry *indexer.Registry, log *slog.Logger) *indexerHandler {
	if log == nil {
		log = slog.Default()
	}
	return &indexerHandler{store: store, registry: registry, log: log}
}

func mountIndexer(r chi.Router, h *indexerHandler) {
	r.Route("/indexer", func(r chi.Router) {
		r.Get("/schema", h.schema)
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toIndexerResource(inst indexer.Instance) indexerResource {
	settings := inst.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("[]")
	}
	return indexerResource{
		ID:                      inst.ID,
		Name:                    inst.Name,
		Implementation:          inst.Implementation,
		Settings:                settings,
		EnableRss:               inst.EnableRss,
		EnableAutomaticSearch:   inst.EnableAutomaticSearch,
		EnableInteractiveSearch: inst.EnableInteractiveSearch,
		Priority:                inst.Priority,
		Added:                   formatTime(inst.Added),
	}
}

func (h *indexerHandler) schema(w http.ResponseWriter, r *http.Request) {
	all := h.registry.All()
	result := make([]indexerSchemaResource, 0, len(all))
	for name, factory := range all {
		impl := factory()
		schema := providers.SchemaFor(impl.Settings())
		fieldsJSON, _ := json.Marshal(schema.Fields)
		result = append(result, indexerSchemaResource{
			Implementation: name,
			Name:           impl.DefaultName(),
			Fields:         json.RawMessage(fieldsJSON),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *indexerHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("v6 indexer list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]indexerResource, 0, len(all))
	for _, inst := range all {
		resources = append(resources, toIndexerResource(inst))
	}
	writeJSON(w, http.StatusOK, resources)
}

// indexerInput is the JSON body for POST/PUT indexer.
type indexerInput struct {
	Name                    string          `json:"name"`
	Implementation          string          `json:"implementation"`
	Settings                json.RawMessage `json:"fields"`
	EnableRss               bool            `json:"enableRss"`
	EnableAutomaticSearch   bool            `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool            `json:"enableInteractiveSearch"`
	Priority                int             `json:"priority"`
}

func (h *indexerHandler) create(w http.ResponseWriter, r *http.Request) {
	var input indexerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}
	settings := input.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	inst, err := h.store.Create(r.Context(), indexer.Instance{
		Name:                    input.Name,
		Implementation:          input.Implementation,
		Settings:                settings,
		EnableRss:               input.EnableRss,
		EnableAutomaticSearch:   input.EnableAutomaticSearch,
		EnableInteractiveSearch: input.EnableInteractiveSearch,
		Priority:                input.Priority,
	})
	if err != nil {
		h.log.Error("v6 indexer create", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toIndexerResource(inst))
}

func (h *indexerHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	inst, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, indexer.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No indexer with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 indexer get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toIndexerResource(inst))
}

func (h *indexerHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	existing, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, indexer.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No indexer with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 indexer update get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input indexerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if len(input.Settings) > 0 {
		existing.Settings = input.Settings
	}
	existing.EnableRss = input.EnableRss
	existing.EnableAutomaticSearch = input.EnableAutomaticSearch
	existing.EnableInteractiveSearch = input.EnableInteractiveSearch
	if input.Priority > 0 {
		existing.Priority = input.Priority
	}

	if err := h.store.Update(r.Context(), existing); err != nil {
		h.log.Error("v6 indexer update", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toIndexerResource(existing))
}

func (h *indexerHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, indexer.ErrNotFound) {
			WriteNotFound(w, r, fmt.Sprintf("No indexer with id %d", id))
			return
		}
		h.log.Error("v6 indexer delete", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
