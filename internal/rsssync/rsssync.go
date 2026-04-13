// Package rsssync implements the RSS sync pipeline: pull releases from all
// enabled indexers, parse each title, match to a known series, evaluate
// against the decision engine, rank per-episode groups, and grab the winner.
package rsssync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/grab"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// groupKey uniquely identifies a (series, season, episode) triple so that
// multiple releases for the same episode can be ranked and the best selected.
type groupKey struct {
	seriesID int64
	season   int
	episode  int
}

// candidate pairs a RemoteEpisode with the original indexer.Release so we can
// pass the full release metadata to GrabService after ranking.
type candidate struct {
	remote    decisionengine.RemoteEpisode
	release   indexer.Release
	episodeID int64
}

// libraryLookup adapts library.SeriesStore to the parser.SeriesLookup interface.
type libraryLookup struct {
	series library.SeriesStore
}

// FindByTitle performs a case-insensitive lookup on title and slug.
func (l *libraryLookup) FindByTitle(ctx context.Context, title string) (int64, bool, error) {
	all, err := l.series.List(ctx)
	if err != nil {
		return 0, false, err
	}
	normalized := strings.ToLower(strings.TrimSpace(title))
	for _, s := range all {
		if strings.ToLower(s.Title) == normalized || strings.ToLower(s.Slug) == normalized {
			return s.ID, true, nil
		}
	}
	return 0, false, nil
}

// Handler is the command handler for the RssSync command. It owns the full
// RSS-to-grab pipeline.
type Handler struct {
	idxStore     indexer.InstanceStore
	idxRegistry  *indexer.Registry
	library      *library.Library
	engine       *decisionengine.Engine
	grabService  *grab.Service
	qualityDefs  profiles.QualityDefinitionStore
	qualityProfs profiles.QualityProfileStore
	cfStore      customformats.Store
	log          *slog.Logger
}

// New constructs a Handler with all required dependencies.
func New(
	idxStore indexer.InstanceStore,
	idxRegistry *indexer.Registry,
	lib *library.Library,
	engine *decisionengine.Engine,
	grabService *grab.Service,
	qualityDefs profiles.QualityDefinitionStore,
	qualityProfs profiles.QualityProfileStore,
	cfStore customformats.Store,
	log *slog.Logger,
) *Handler {
	return &Handler{
		idxStore:     idxStore,
		idxRegistry:  idxRegistry,
		library:      lib,
		engine:       engine,
		grabService:  grabService,
		qualityDefs:  qualityDefs,
		qualityProfs: qualityProfs,
		cfStore:      cfStore,
		log:          log,
	}
}

// Handle implements commands.Handler. It runs the full RSS sync pipeline:
// fetch → parse → match → evaluate → rank → grab.
func (h *Handler) Handle(ctx context.Context, cmd commands.Command) error {
	// ---- Step 1: collect all releases from enabled RSS indexers --------

	instances, err := h.idxStore.List(ctx)
	if err != nil {
		return fmt.Errorf("rsssync: list indexers: %w", err)
	}

	var allReleases []indexer.Release
	for _, inst := range instances {
		if !inst.EnableRss {
			continue
		}
		factory, err := h.idxRegistry.Get(inst.Implementation)
		if err != nil {
			h.log.WarnContext(ctx, "no factory for indexer", "implementation", inst.Implementation, "id", inst.ID)
			continue
		}
		idx := factory()
		if len(inst.Settings) > 0 {
			if err := json.Unmarshal(inst.Settings, idx.Settings()); err != nil {
				h.log.WarnContext(ctx, "failed to unmarshal indexer settings",
					"implementation", inst.Implementation, "id", inst.ID, "error", err)
				continue
			}
		}
		releases, err := idx.FetchRss(ctx)
		if err != nil {
			h.log.WarnContext(ctx, "FetchRss failed", "indexer", inst.Name, "error", err)
			continue
		}
		allReleases = append(allReleases, releases...)
	}

	if len(allReleases) == 0 {
		h.log.InfoContext(ctx, "rsssync: no releases found")
		return nil
	}

	// ---- Step 2: load shared data needed for every release ---------------

	allDefs, err := h.qualityDefs.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("rsssync: get quality definitions: %w", err)
	}

	allCFs, err := h.cfStore.List(ctx)
	if err != nil {
		return fmt.Errorf("rsssync: list custom formats: %w", err)
	}

	// Use quality profile ID 1 (the default "Any" profile seeded in M5).
	defaultProfile, err := h.qualityProfs.GetByID(ctx, 1)
	if err != nil {
		h.log.WarnContext(ctx, "rsssync: could not load quality profile 1, using empty profile", "error", err)
		defaultProfile = profiles.QualityProfile{ID: 1, Name: "Any"}
	}

	ll := &libraryLookup{series: h.library.Series}

	// ---- Step 3: process each release ------------------------------------

	// groups maps groupKey → list of candidates accepted by the engine.
	groups := make(map[groupKey][]candidate)

	for _, rel := range allReleases {
		parsed := parser.ParseTitle(rel.Title)
		if parsed.SeriesTitle == "" {
			h.log.DebugContext(ctx, "rsssync: could not parse series title, skipping", "title", rel.Title)
			continue
		}

		// Match to a library series.
		seriesID, found, err := ll.FindByTitle(ctx, parsed.SeriesTitle)
		if err != nil {
			h.log.WarnContext(ctx, "rsssync: series lookup error", "title", parsed.SeriesTitle, "error", err)
			continue
		}
		if !found {
			h.log.DebugContext(ctx, "rsssync: no series match, skipping", "parsed", parsed.SeriesTitle)
			continue
		}

		if len(parsed.EpisodeNumbers) == 0 {
			h.log.DebugContext(ctx, "rsssync: no episode numbers parsed, skipping", "title", rel.Title)
			continue
		}

		// Find matching episodes in the library.
		allEps, err := h.library.Episodes.ListForSeries(ctx, seriesID)
		if err != nil {
			h.log.WarnContext(ctx, "rsssync: list episodes failed", "seriesID", seriesID, "error", err)
			continue
		}

		for _, epNum := range parsed.EpisodeNumbers {
			epNum := epNum
			season := parsed.SeasonNumber

			var matchedEp *library.Episode
			for i := range allEps {
				ep := &allEps[i]
				if int(ep.SeasonNumber) == season && int(ep.EpisodeNumber) == epNum {
					matchedEp = ep
					break
				}
			}
			if matchedEp == nil {
				h.log.DebugContext(ctx, "rsssync: episode not found in library",
					"seriesID", seriesID, "season", season, "episode", epNum)
				continue
			}

			// Skip if episode is not monitored.
			if !matchedEp.Monitored {
				h.log.DebugContext(ctx, "rsssync: episode not monitored, skipping",
					"episodeID", matchedEp.ID)
				continue
			}

			// Resolve quality ID.
			qualityID := resolveQualityID(parsed.Quality, allDefs)

			// Score custom formats.
			cfIDs := matchedCFIDs(parsed, allCFs)
			cfScore := customformats.Score(parsed, allCFs, defaultProfile)

			// Build RemoteEpisode for the decision engine.
			remote := decisionengine.RemoteEpisode{
				Release: decisionengine.Release{
					Title:    rel.Title,
					Size:     rel.Size,
					Indexer:  rel.Indexer,
					Protocol: string(rel.Protocol),
				},
				ParsedInfo:    parsed,
				SeriesID:      seriesID,
				EpisodeIDs:    []int64{matchedEp.ID},
				Quality:       parsed.Quality,
				QualityID:     qualityID,
				CustomFormats: cfIDs,
				CFScore:       cfScore,
			}

			decision, rejections := h.engine.Evaluate(ctx, remote, defaultProfile)
			if decision == decisionengine.Reject {
				reasons := make([]string, len(rejections))
				for i, r := range rejections {
					reasons[i] = r.Reason
				}
				h.log.DebugContext(ctx, "rsssync: release rejected",
					"title", rel.Title, "reasons", reasons)
				continue
			}

			key := groupKey{seriesID: seriesID, season: season, episode: epNum}
			groups[key] = append(groups[key], candidate{
				remote:    remote,
				release:   rel,
				episodeID: matchedEp.ID,
			})
		}
	}

	// ---- Step 4: rank each group, pick the best, grab -------------------

	for key, candidates := range groups {
		// Build slice of RemoteEpisodes for ranking.
		remotes := make([]decisionengine.RemoteEpisode, len(candidates))
		for i, c := range candidates {
			remotes[i] = c.remote
		}
		ranked := h.engine.Rank(remotes, defaultProfile)
		if len(ranked) == 0 {
			continue
		}
		best := ranked[0]

		// Find the original candidate entry that matches the ranked winner.
		var winner candidate
		for _, c := range candidates {
			if c.remote.Release.Title == best.Release.Title &&
				c.remote.CFScore == best.CFScore {
				winner = c
				break
			}
		}

		// Resolve quality name for history.
		qualityName := qualityNameForID(best.QualityID, allDefs)

		if err := h.grabService.Grab(
			ctx,
			winner.release,
			key.seriesID,
			[]int64{winner.episodeID},
			qualityName,
		); err != nil {
			h.log.WarnContext(ctx, "rsssync: grab failed",
				"title", winner.release.Title, "error", err)
		}
	}

	return nil
}

// resolveQualityID maps a ParsedQuality to its QualityDefinition ID.
// Returns 0 if no matching definition is found.
func resolveQualityID(parsed parser.ParsedQuality, defs []profiles.QualityDefinition) int {
	for _, d := range defs {
		if d.Source == string(parsed.Source) && d.Resolution == string(parsed.Resolution) {
			return d.ID
		}
	}
	return 0
}

// qualityNameForID returns the name of the quality definition with the given ID,
// or an empty string if not found.
func qualityNameForID(id int, defs []profiles.QualityDefinition) string {
	for _, d := range defs {
		if d.ID == id {
			return d.Name
		}
	}
	return ""
}

// matchedCFIDs returns the IDs of all custom formats that match the parsed info.
func matchedCFIDs(parsed parser.ParsedEpisodeInfo, formats []customformats.CustomFormat) []int {
	var ids []int
	for _, cf := range formats {
		if customformats.Match(parsed, cf) {
			ids = append(ids, cf.ID)
		}
	}
	return ids
}
