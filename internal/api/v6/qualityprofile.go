package v6

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/profiles"
)

// qualityProfileResource is the v6 JSON shape for a quality profile.
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

// qualityProfileInput is the JSON body for POST/PUT quality profile.
type qualityProfileInput struct {
	Name              string `json:"name"`
	UpgradeAllowed    bool   `json:"upgradeAllowed"`
	Cutoff            int    `json:"cutoff"`
	MinFormatScore    int    `json:"minFormatScore"`
	CutoffFormatScore int    `json:"cutoffFormatScore"`
	Items             []struct {
		QualityID int  `json:"qualityId"`
		Allowed   bool `json:"allowed"`
	} `json:"items"`
}

type qualityProfileHandler struct {
	store profiles.QualityProfileStore
	defs  profiles.QualityDefinitionStore
	log   *slog.Logger
}

func newQualityProfileHandler(store profiles.QualityProfileStore, defs profiles.QualityDefinitionStore, log *slog.Logger) *qualityProfileHandler {
	if log == nil {
		log = slog.Default()
	}
	return &qualityProfileHandler{store: store, defs: defs, log: log}
}

func mountQualityProfile(r chi.Router, h *qualityProfileHandler) {
	r.Route("/qualityprofile", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
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
		h.log.Error("v6 qualityprofile list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resources := make([]qualityProfileResource, 0, len(all))
	for _, p := range all {
		resources = append(resources, h.toResource(p))
	}
	writeJSON(w, http.StatusOK, Page[qualityProfileResource]{
		Data: resources,
		Pagination: Pagination{
			Limit:   len(resources),
			HasMore: false,
		},
	})
}

func (h *qualityProfileHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	p, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, profiles.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No quality profile with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 qualityprofile get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, h.toResource(p))
}

func (h *qualityProfileHandler) create(w http.ResponseWriter, r *http.Request) {
	var input qualityProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	items := make([]profiles.QualityProfileItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, profiles.QualityProfileItem{
			QualityID: item.QualityID,
			Allowed:   item.Allowed,
		})
	}

	p, err := h.store.Create(r.Context(), profiles.QualityProfile{
		Name:              input.Name,
		UpgradeAllowed:    input.UpgradeAllowed,
		Cutoff:            input.Cutoff,
		MinFormatScore:    input.MinFormatScore,
		CutoffFormatScore: input.CutoffFormatScore,
		Items:             items,
	})
	if err != nil {
		h.log.Error("v6 qualityprofile create", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, h.toResource(p))
}

func (h *qualityProfileHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	existing, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, profiles.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No quality profile with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 qualityprofile update get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input qualityProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	existing.UpgradeAllowed = input.UpgradeAllowed
	if input.Cutoff > 0 {
		existing.Cutoff = input.Cutoff
	}
	existing.MinFormatScore = input.MinFormatScore
	existing.CutoffFormatScore = input.CutoffFormatScore

	if err := h.store.Update(r.Context(), existing); err != nil {
		h.log.Error("v6 qualityprofile update", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, h.toResource(existing))
}

func (h *qualityProfileHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, profiles.ErrNotFound) {
			WriteNotFound(w, r, fmt.Sprintf("No quality profile with id %d", id))
			return
		}
		h.log.Error("v6 qualityprofile delete", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
