// Package metadata provides pluggable "consumers" that emit .nfo / image /
// extras files alongside media files for third-party players (Kodi, Plex,
// Roksbox, WDTV). Consumers are invoked on episode-file import and series
// refresh events.
//
// Ported architecturally from Sonarr (src/NzbDrone.Core/Extras/Metadata/).
package metadata

import (
	"context"
	"errors"
)

// ErrNotFound is returned for unknown consumer instances or configs.
var ErrNotFound = errors.New("metadata: not found")

// SeriesInfo carries the subset of series data relevant to metadata emission.
type SeriesInfo struct {
	ID       int64
	Title    string
	Path     string
	TvdbID   int64
	ImdbID   string
	Year     int
	Overview string
	Runtime  int // minutes
	Network  string
	Status   string
	Genres   []string
	Actors   []Actor
	PosterURL  string
	FanartURL  string
	BannerURL  string
	Certification string
}

// Actor is one cast-list entry.
type Actor struct {
	Name      string
	Role      string
	Order     int
	ThumbURL  string
}

// EpisodeInfo carries the subset of episode data relevant to metadata.
type EpisodeInfo struct {
	ID            int64
	SeriesID      int64
	SeasonNumber  int
	EpisodeNumber int
	Title         string
	Overview      string
	AirDate       string // YYYY-MM-DD
	Runtime       int
	ScreenshotURL string
}

// EpisodeFileInfo identifies the on-disk media file a consumer should emit
// sidecar metadata next to.
type EpisodeFileInfo struct {
	Path         string // absolute path to the video file
	RelativePath string
	SeriesPath   string // root of the series folder
	Quality      string
	ReleaseGroup string
	Size         int64
}

// Context bundles a series + its episodes + a pointer to the file being
// imported or refreshed. Consumers receive this on each hook invocation.
type Context struct {
	Series      SeriesInfo
	Episode     EpisodeInfo
	EpisodeFile EpisodeFileInfo
}

// Consumer emits metadata sidecar files (nfo / xml / jpg) next to episode and
// series files. Implementations are registered by Implementation identifier.
type Consumer interface {
	Implementation() string
	DefaultName() string
	Settings() any
	OnEpisodeFileImport(ctx context.Context, c Context) error
	OnSeriesRefresh(ctx context.Context, s SeriesInfo) error
}

// Registry holds Consumer constructors by Implementation.
type Registry struct {
	builders map[string]func() Consumer
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry { return &Registry{builders: map[string]func() Consumer{}} }

// Register adds a Consumer constructor under the given Implementation name.
func (r *Registry) Register(implementation string, build func() Consumer) {
	r.builders[implementation] = build
}

// Names returns all registered Consumer identifiers.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.builders))
	for name := range r.builders {
		out = append(out, name)
	}
	return out
}

// Build returns a fresh Consumer for the given implementation, or
// ErrNotFound if unknown.
func (r *Registry) Build(implementation string) (Consumer, error) {
	build, ok := r.builders[implementation]
	if !ok {
		return nil, ErrNotFound
	}
	return build(), nil
}
