package v3

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/profiles"
)

// qualityProfileResource is the Sonarr v3 JSON shape for a quality profile.
type qualityProfileResource struct {
	ID                int                          `json:"id"`
	Name              string                       `json:"name"`
	UpgradeAllowed    bool                         `json:"upgradeAllowed"`
	Cutoff            int                          `json:"cutoff"`
	Items             []qualityProfileItemResource `json:"items"`
	MinFormatScore    int                          `json:"minFormatScore"`
	CutoffFormatScore int                          `json:"cutoffFormatScore"`
	FormatItems       []any                        `json:"formatItems"`
}

// qualityProfileItemResource is one item in the quality profile items list.
type qualityProfileItemResource struct {
	Quality qualityInProfile `json:"quality"`
	Items   []any            `json:"items"`
	Allowed bool             `json:"allowed"`
}

// qualityInProfile is the nested quality object inside a profile item.
type qualityInProfile struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Resolution string `json:"resolution"`
}

// qualityProfileHandler handles /api/v3/qualityprofile endpoints.
type qualityProfileHandler struct {
	store profiles.QualityProfileStore
	defs  profiles.QualityDefinitionStore
	log   *slog.Logger
}

// NewQualityProfileHandler constructs a qualityProfileHandler.
func NewQualityProfileHandler(store profiles.QualityProfileStore, defs profiles.QualityDefinitionStore, log *slog.Logger) *qualityProfileHandler {
	return &qualityProfileHandler{store: store, defs: defs, log: log}
}

// MountQualityProfile registers /api/v3/qualityprofile routes.
func MountQualityProfile(r chi.Router, h *qualityProfileHandler) {
	r.Route("/api/v3/qualityprofile", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
	})
}

func (h *qualityProfileHandler) toResource(p profiles.QualityProfile) qualityProfileResource {
	items := make([]qualityProfileItemResource, 0, len(p.Items))
	for _, item := range p.Items {
		items = append(items, qualityProfileItemResource{
			Quality: qualityInProfile{
				ID:   item.QualityID,
				Name: "",
			},
			Items:   []any{},
			Allowed: item.Allowed,
		})
	}
	return qualityProfileResource{
		ID:                p.ID,
		Name:              p.Name,
		UpgradeAllowed:    p.UpgradeAllowed,
		Cutoff:            p.Cutoff,
		Items:             items,
		MinFormatScore:    p.MinFormatScore,
		CutoffFormatScore: p.CutoffFormatScore,
		FormatItems:       []any{},
	}
}

func (h *qualityProfileHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("qualityprofile list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]qualityProfileResource, 0, len(all))
	for _, p := range all {
		resources = append(resources, h.toResource(p))
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *qualityProfileHandler) get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	p, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, profiles.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("qualityprofile get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, h.toResource(p))
}
