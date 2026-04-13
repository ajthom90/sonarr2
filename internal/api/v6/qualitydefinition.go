package v6

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/profiles"
)

// qualityDefinitionResource is the v6 JSON shape for a quality definition.
type qualityDefinitionResource struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Source        string  `json:"source"`
	Resolution    string  `json:"resolution"`
	MinSize       float64 `json:"minSize"`
	MaxSize       float64 `json:"maxSize"`
	PreferredSize float64 `json:"preferredSize"`
}

type qualityDefinitionHandler struct {
	store profiles.QualityDefinitionStore
	log   *slog.Logger
}

func newQualityDefinitionHandler(store profiles.QualityDefinitionStore, log *slog.Logger) *qualityDefinitionHandler {
	if log == nil {
		log = slog.Default()
	}
	return &qualityDefinitionHandler{store: store, log: log}
}

func mountQualityDefinition(r chi.Router, h *qualityDefinitionHandler) {
	r.Get("/qualitydefinition", h.list)
}

func (h *qualityDefinitionHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.GetAll(r.Context())
	if err != nil {
		h.log.Error("v6 qualitydefinition list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]qualityDefinitionResource, 0, len(all))
	for _, d := range all {
		resources = append(resources, qualityDefinitionResource{
			ID:            d.ID,
			Name:          d.Name,
			Source:        d.Source,
			Resolution:    d.Resolution,
			MinSize:       d.MinSize,
			MaxSize:       d.MaxSize,
			PreferredSize: d.PreferredSize,
		})
	}
	writeJSON(w, http.StatusOK, resources)
}
