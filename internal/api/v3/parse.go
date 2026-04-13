package v3

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/parser"
)

// parseResultResource is the JSON shape returned by /api/v3/parse.
type parseResultResource struct {
	Title             string             `json:"title"`
	ParsedEpisodeInfo parsedInfoResource `json:"parsedEpisodeInfo"`
}

// parsedInfoResource maps the parsed episode info to Sonarr's JSON shape.
type parsedInfoResource struct {
	ReleasedTitle          string `json:"releaseTitle"`
	SeriesTitle            string `json:"seriesTitle"`
	SeriesType             string `json:"seriesType"`
	SeasonNumber           int    `json:"seasonNumber"`
	EpisodeNumbers         []int  `json:"episodeNumbers"`
	AbsoluteEpisodeNumbers []int  `json:"absoluteEpisodeNumbers"`
	ReleaseGroup           string `json:"releaseGroup"`
	Quality                any    `json:"quality"`
	Languages              []any  `json:"languages"`
	FullSeason             bool   `json:"fullSeason"`
}

// MountParse registers /api/v3/parse routes.
func MountParse(r chi.Router) {
	r.Route("/api/v3/parse", func(r chi.Router) {
		r.Get("/", parseHandler)
	})
}

func parseHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		writeError(w, http.StatusBadRequest, "title query parameter is required")
		return
	}

	info := parser.ParseTitle(title)

	episodeNums := info.EpisodeNumbers
	if episodeNums == nil {
		episodeNums = []int{}
	}
	absNums := info.AbsoluteEpisodeNumbers
	if absNums == nil {
		absNums = []int{}
	}

	result := parseResultResource{
		Title: title,
		ParsedEpisodeInfo: parsedInfoResource{
			ReleasedTitle:          info.ReleaseTitle,
			SeriesTitle:            info.SeriesTitle,
			SeriesType:             string(info.SeriesType),
			SeasonNumber:           info.SeasonNumber,
			EpisodeNumbers:         episodeNums,
			AbsoluteEpisodeNumbers: absNums,
			ReleaseGroup:           info.ReleaseGroup,
			Quality: map[string]any{
				"quality": map[string]any{
					"source":     string(info.Quality.Source),
					"resolution": string(info.Quality.Resolution),
					"modifier":   string(info.Quality.Modifier),
				},
				"revision": map[string]any{
					"version": info.Quality.Revision,
				},
			},
			Languages:  []any{},
			FullSeason: false,
		},
	}
	writeJSON(w, http.StatusOK, result)
}
