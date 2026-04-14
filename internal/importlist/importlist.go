// Package importlist implements Sonarr's Import Lists feature: pluggable
// list providers (Trakt, Plex Watchlist, AniList, MyAnimeList, Simkl,
// another Sonarr instance, generic RSS, Custom) that auto-populate the
// series library on a schedule.
//
// A provider's Fetch method returns []Item (TVDB IDs + titles) which the
// sync loop then filters through ImportListExclusion and adds to the
// library according to ShouldMonitor preset and quality profile.
//
// Ported architecturally from Sonarr (src/NzbDrone.Core/ImportLists/).
package importlist

import (
	"context"
	"errors"

	"github.com/ajthom90/sonarr2/internal/providers"
)

// ErrNotFound is returned for missing instances or unknown implementations.
var ErrNotFound = errors.New("importlist: not found")

// Monitor enumerates Sonarr's "monitoring presets" applied to newly-added
// series from this list.
type Monitor string

const (
	MonitorAll          Monitor = "all"
	MonitorFuture       Monitor = "future"
	MonitorMissing      Monitor = "missing"
	MonitorExisting     Monitor = "existing"
	MonitorPilot        Monitor = "pilot"
	MonitorFirstSeason  Monitor = "firstSeason"
	MonitorLatestSeason Monitor = "latestSeason"
	MonitorNone         Monitor = "none"
)

// Item is one entry returned by a list provider.
type Item struct {
	TvdbID int64
	Title  string
	Year   int
}

// ListProvider is the pluggable surface each import-list provider implements.
// Fetch runs on the sync interval and returns the current list. Test() is
// called from the /test endpoint to validate credentials.
type ListProvider interface {
	providers.Provider
	Fetch(ctx context.Context) ([]Item, error)
	Test(ctx context.Context) error
}

// Instance is a saved import-list row.
type Instance struct {
	ID                     int
	Name                   string
	Implementation         string
	Settings               []byte // provider-specific JSON
	EnableAutomaticAdd     bool
	ShouldMonitor          Monitor
	ShouldMonitorExisting  bool
	ShouldSearch           bool
	RootFolderPath         string
	QualityProfileID       int
	SeriesType             string // standard|daily|anime
	SeasonFolder           bool
	Tags                   []int
	ListType               string // program|advanced|other
	MinRefreshIntervalMins int
}

// Exclusion is a TVDB ID users never want auto-added.
type Exclusion struct {
	ID     int
	TvdbID int64
	Title  string
}

// Store provides CRUD for import-list instances and exclusions.
type Store interface {
	Create(ctx context.Context, ins Instance) (Instance, error)
	GetByID(ctx context.Context, id int) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	Update(ctx context.Context, ins Instance) error
	Delete(ctx context.Context, id int) error

	CreateExclusion(ctx context.Context, ex Exclusion) (Exclusion, error)
	ListExclusions(ctx context.Context) ([]Exclusion, error)
	DeleteExclusion(ctx context.Context, id int) error
	IsExcluded(ctx context.Context, tvdbID int64) (bool, error)
}

// Registry holds ListProvider constructors keyed by Implementation.
type Registry struct {
	builders map[string]func() ListProvider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry { return &Registry{builders: map[string]func() ListProvider{}} }

// Register adds a provider constructor under the given identifier.
func (r *Registry) Register(impl string, build func() ListProvider) { r.builders[impl] = build }

// Names lists all registered identifiers (used for /schema responses).
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.builders))
	for k := range r.builders {
		out = append(out, k)
	}
	return out
}

// Build returns a fresh provider for the implementation, or ErrNotFound.
func (r *Registry) Build(impl string) (ListProvider, error) {
	b, ok := r.builders[impl]
	if !ok {
		return nil, ErrNotFound
	}
	return b(), nil
}
