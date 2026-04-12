package decisionengine

import (
	"context"
	"sort"

	"github.com/ajthom90/sonarr2/internal/profiles"
)

// Engine runs a series of Spec implementations against a release and decides
// whether it should be accepted, and how it ranks among accepted releases.
type Engine struct {
	specs []Spec
}

// New constructs an Engine with the provided specs. Specs are evaluated in
// the order they are provided.
func New(specs ...Spec) *Engine {
	return &Engine{specs: specs}
}

// Evaluate runs every spec against the release. ALL specs are evaluated even
// after the first rejection so that the UI can display a complete list of
// reasons a release was not grabbed. Returns Reject if any spec rejected,
// Accept otherwise.
func (e *Engine) Evaluate(ctx context.Context, remote RemoteEpisode, profile profiles.QualityProfile) (Decision, []Rejection) {
	var rejections []Rejection
	for _, s := range e.specs {
		d, r := s.Evaluate(ctx, remote, profile)
		if d == Reject {
			rejections = append(rejections, r...)
		}
	}
	if len(rejections) > 0 {
		return Reject, rejections
	}
	return Accept, nil
}

// Rank sorts a slice of RemoteEpisodes by preference (highest preference
// first). The sort order is:
//  1. Custom format score — higher is better.
//  2. Quality — lower index in profile.Items means higher preference.
//  3. Size — larger is better (tiebreaker).
//
// sort.SliceStable is used so that releases with identical scores remain in
// their original (insertion) order relative to each other.
func (e *Engine) Rank(remotes []RemoteEpisode, profile profiles.QualityProfile) []RemoteEpisode {
	sort.SliceStable(remotes, func(i, j int) bool {
		if remotes[i].CFScore != remotes[j].CFScore {
			return remotes[i].CFScore > remotes[j].CFScore
		}
		qi := qualityIndex(remotes[i], profile)
		qj := qualityIndex(remotes[j], profile)
		if qi != qj {
			return qi < qj
		}
		return remotes[i].Release.Size > remotes[j].Release.Size
	})
	return remotes
}

// qualityIndex returns the position of the remote's quality in the profile's
// Items slice. Lower index means higher preference. If the quality is not
// found in the profile, len(profile.Items) is returned (worst rank).
func qualityIndex(remote RemoteEpisode, profile profiles.QualityProfile) int {
	for i, item := range profile.Items {
		if item.QualityID == remote.QualityID {
			return i
		}
	}
	return len(profile.Items)
}
