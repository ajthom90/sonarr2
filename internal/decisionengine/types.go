// Package decisionengine evaluates releases against quality profiles and ranks
// surviving releases by preference. It is composed of a chain of Spec
// implementations, each of which can Accept or Reject a release with a reason.
package decisionengine

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// Decision is the outcome of evaluating a release against a spec.
type Decision int

const (
	// Accept means the spec approves the release (or has no opinion).
	Accept Decision = iota
	// Reject means the spec disapproves the release.
	Reject
)

// RejectionType classifies whether a rejection is permanent or transient.
type RejectionType int

const (
	// Permanent means the release can never be accepted (e.g. wrong quality).
	Permanent RejectionType = iota
	// Temporary means the release might be acceptable later (e.g. not yet seeded).
	Temporary
)

// Rejection captures the reason a spec rejected a release.
type Rejection struct {
	Type   RejectionType
	Reason string
	Spec   string // name of the spec that produced this rejection
}

// Release is the raw metadata for a candidate release from an indexer.
type Release struct {
	Title    string
	Size     int64 // bytes
	Indexer  string
	Age      int    // days (usenet) or minutes (torrent)
	Protocol string // "usenet" or "torrent"
	Seeders  int
}

// RemoteEpisode pairs a Release with the parsed information and context needed
// by the decision engine specs.
type RemoteEpisode struct {
	Release       Release
	ParsedInfo    parser.ParsedEpisodeInfo
	SeriesID      int64
	EpisodeIDs    []int64
	Quality       parser.ParsedQuality
	QualityID     int   // profile quality definition ID for this release's quality
	CustomFormats []int // matched custom format IDs
	CFScore       int

	// Upgrade context — zero values mean no existing file.
	ExistingFileQualityID    int    // 0 = no existing file
	ExistingFileReleaseGroup string // release group of the existing file
}

// Spec is a single evaluation rule. Each spec is called with the release and
// the active quality profile and returns a decision and any rejections.
type Spec interface {
	Name() string
	Evaluate(ctx context.Context, remote RemoteEpisode, profile profiles.QualityProfile) (Decision, []Rejection)
}
