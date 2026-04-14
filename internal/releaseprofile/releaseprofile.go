// Package releaseprofile implements Sonarr's Release Profile feature.
//
// A release profile filters releases by term match on the release title,
// with separate "required" and "ignored" term lists. An optional indexer
// scope (IndexerID=0 means "all") and optional tag list bind the profile
// to specific series.
//
// Applied by the decision engine: a release must contain ALL required terms
// (case-insensitive substring match by default; /pattern/ enables regex), and
// MUST NOT contain ANY ignored term. Release profiles support Sonarr's
// "Must Contain / Must Not Contain" functionality.
//
// Ported from Sonarr (src/NzbDrone.Core/Profiles/Releases/).
package releaseprofile

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// ErrNotFound is returned when a release profile is missing.
var ErrNotFound = errors.New("releaseprofile: not found")

// Profile represents one release profile row.
type Profile struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Required  []string `json:"required"`
	Ignored   []string `json:"ignored"`
	IndexerID int      `json:"indexerId"`
	Tags      []int    `json:"tags"`
}

// Store provides CRUD access to release profiles.
type Store interface {
	Create(ctx context.Context, p Profile) (Profile, error)
	GetByID(ctx context.Context, id int) (Profile, error)
	List(ctx context.Context) ([]Profile, error)
	Update(ctx context.Context, p Profile) error
	Delete(ctx context.Context, id int) error
}

// Match reports whether releaseTitle satisfies the profile: all Required terms
// present and no Ignored term present. Case-insensitive. A term wrapped in
// "/regex/" is treated as a regex (matches Sonarr's ReleaseProfilePreferred
// regex support).
func Match(p Profile, releaseTitle string) bool {
	for _, term := range p.Required {
		if !termMatches(term, releaseTitle) {
			return false
		}
	}
	for _, term := range p.Ignored {
		if termMatches(term, releaseTitle) {
			return false
		}
	}
	return true
}

func termMatches(term, title string) bool {
	if term == "" {
		return true
	}
	if len(term) >= 2 && strings.HasPrefix(term, "/") && strings.HasSuffix(term, "/") {
		re, err := regexp.Compile("(?i)" + term[1:len(term)-1])
		if err != nil {
			return false
		}
		return re.MatchString(title)
	}
	return strings.Contains(strings.ToLower(title), strings.ToLower(term))
}
