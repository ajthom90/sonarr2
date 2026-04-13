package v3

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/profiles"
)

// qualityDefinitionResource is the Sonarr v3 JSON shape for a quality definition.
type qualityDefinitionResource struct {
	ID            int                    `json:"id"`
	Quality       qualityDefinitionInner `json:"quality"`
	Title         string                 `json:"title"`
	MinSize       float64                `json:"minSize"`
	MaxSize       float64                `json:"maxSize"`
	PreferredSize float64                `json:"preferredSize"`
}

// qualityDefinitionInner matches the nested quality object Sonarr uses.
type qualityDefinitionInner struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Resolution string `json:"resolution"`
}

// qualityDefinitionHandler handles /api/v3/qualitydefinition endpoints.
type qualityDefinitionHandler struct {
	store profiles.QualityDefinitionStore
	log   *slog.Logger
}

// NewQualityDefinitionHandler constructs a qualityDefinitionHandler.
func NewQualityDefinitionHandler(store profiles.QualityDefinitionStore, log *slog.Logger) *qualityDefinitionHandler {
	return &qualityDefinitionHandler{store: store, log: log}
}

// MountQualityDefinition registers /api/v3/qualitydefinition routes.
func MountQualityDefinition(r chi.Router, h *qualityDefinitionHandler) {
	r.Route("/api/v3/qualitydefinition", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

func (h *qualityDefinitionHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.GetAll(r.Context())
	if err != nil {
		h.log.Error("qualitydefinition list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]qualityDefinitionResource, 0, len(all))
	for _, d := range all {
		resources = append(resources, qualityDefinitionResource{
			ID: d.ID,
			Quality: qualityDefinitionInner{
				ID:         d.ID,
				Name:       d.Name,
				Source:     d.Source,
				Resolution: d.Resolution,
			},
			Title:         d.Name,
			MinSize:       d.MinSize,
			MaxSize:       d.MaxSize,
			PreferredSize: d.PreferredSize,
		})
	}
	writeJSON(w, http.StatusOK, resources)
}
