package v6

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/customformats"
)

// customFormatResource is the v6 JSON shape for a custom format.
type customFormatResource struct {
	ID                  int                        `json:"id"`
	Name                string                     `json:"name"`
	IncludeWhenRenaming bool                       `json:"includeCustomFormatWhenRenaming"`
	Specifications      []customFormatSpecResource `json:"specifications"`
}

// customFormatSpecResource is one specification within a custom format.
type customFormatSpecResource struct {
	Name           string                  `json:"name"`
	Implementation string                  `json:"implementation"`
	Negate         bool                    `json:"negate"`
	Required       bool                    `json:"required"`
	Fields         []customFormatSpecField `json:"fields"`
}

// customFormatSpecField is a field within a specification.
type customFormatSpecField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type customFormatHandler struct {
	store customformats.Store
	log   *slog.Logger
}

func newCustomFormatHandler(store customformats.Store, log *slog.Logger) *customFormatHandler {
	if log == nil {
		log = slog.Default()
	}
	return &customFormatHandler{store: store, log: log}
}

func mountCustomFormat(r chi.Router, h *customFormatHandler) {
	r.Route("/customformat", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
	})
}

func toCustomFormatResource(cf customformats.CustomFormat) customFormatResource {
	specs := make([]customFormatSpecResource, 0, len(cf.Specifications))
	for _, s := range cf.Specifications {
		specs = append(specs, customFormatSpecResource{
			Name:           s.Name,
			Implementation: s.Implementation,
			Negate:         s.Negate,
			Required:       s.Required,
			Fields: []customFormatSpecField{
				{Name: "value", Value: s.Value},
			},
		})
	}
	return customFormatResource{
		ID:                  cf.ID,
		Name:                cf.Name,
		IncludeWhenRenaming: cf.IncludeWhenRenaming,
		Specifications:      specs,
	}
}

func (h *customFormatHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("v6 customformat list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]customFormatResource, 0, len(all))
	for _, cf := range all {
		resources = append(resources, toCustomFormatResource(cf))
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *customFormatHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	cf, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, customformats.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No custom format with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 customformat get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toCustomFormatResource(cf))
}
