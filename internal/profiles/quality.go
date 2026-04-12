// Package profiles owns the quality definition and quality profile domain for
// sonarr2. Quality definitions describe the size/bitrate expectations for a
// given source+resolution combination; quality profiles combine an ordered list
// of acceptable qualities with optional custom-format score gates.
package profiles

import "context"

// QualityDefinition describes the expected file-size range for a given quality
// tier (source + resolution combination).
type QualityDefinition struct {
	ID            int
	Name          string
	Source        string  // e.g. "webdl", "bluray" — matches parser.QualitySource
	Resolution    string  // e.g. "1080p" — matches parser.Resolution
	MinSize       float64 // MB per minute of runtime
	MaxSize       float64
	PreferredSize float64
}

// QualityProfileItem is one entry in the ordered quality list of a profile.
// The order in the slice determines preference (index 0 = highest preference).
type QualityProfileItem struct {
	QualityID int
	Allowed   bool
}

// FormatScoreItem is the weight assigned to one custom format within a profile.
type FormatScoreItem struct {
	FormatID int
	Score    int
}

// QualityProfile is a named set of rules that govern which releases are
// acceptable and how they are ranked.
type QualityProfile struct {
	ID                int
	Name              string
	UpgradeAllowed    bool
	Cutoff            int // quality definition ID that is "good enough" to stop upgrading
	Items             []QualityProfileItem
	MinFormatScore    int
	CutoffFormatScore int
	FormatItems       []FormatScoreItem
}

// QualityDefinitionStore reads and updates quality definitions.
// Definitions are seeded by migrations and are not user-created; only the
// size ranges can be adjusted.
type QualityDefinitionStore interface {
	GetAll(ctx context.Context) ([]QualityDefinition, error)
	GetByID(ctx context.Context, id int) (QualityDefinition, error)
}

// QualityProfileStore provides full CRUD for quality profiles.
type QualityProfileStore interface {
	Create(ctx context.Context, p QualityProfile) (QualityProfile, error)
	GetByID(ctx context.Context, id int) (QualityProfile, error)
	List(ctx context.Context) ([]QualityProfile, error)
	Update(ctx context.Context, p QualityProfile) error
	Delete(ctx context.Context, id int) error
}
