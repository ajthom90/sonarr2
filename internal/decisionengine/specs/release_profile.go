// SPDX-License-Identifier: GPL-3.0-or-later
// Release profile (Must Contain / Must Not Contain) decision spec.

package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/releaseprofile"
)

// ReleaseProfileSpec rejects releases that fail to satisfy any enabled
// release profile's Required or Ignored term lists. A profile scoped to a
// specific indexer only applies when the release originates from that
// indexer; a profile with tags only applies when the series carries one of
// the tags (caller supplies via ReleaseProfilesForSeries).
type ReleaseProfileSpec struct {
	// ProfilesFn returns the set of enabled release profiles that apply to
	// the given series + indexer context. Nil = no profiles to evaluate.
	ProfilesFn func(ctx context.Context, seriesID int64, indexerName string) ([]releaseprofile.Profile, error)
}

// Name implements decisionengine.Spec.
func (s ReleaseProfileSpec) Name() string { return "ReleaseProfile" }

// Evaluate implements decisionengine.Spec.
func (s ReleaseProfileSpec) Evaluate(ctx context.Context, remote decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if s.ProfilesFn == nil {
		return decisionengine.Accept, nil
	}
	profs, err := s.ProfilesFn(ctx, remote.SeriesID, remote.Release.Indexer)
	if err != nil || len(profs) == 0 {
		return decisionengine.Accept, nil
	}
	for _, p := range profs {
		if !p.Enabled {
			continue
		}
		if !releaseprofile.Match(p, remote.Release.Title) {
			return decisionengine.Reject, []decisionengine.Rejection{{
				Type:   decisionengine.Permanent,
				Reason: fmt.Sprintf("Release profile %q rejected release %q", p.Name, remote.Release.Title),
				Spec:   s.Name(),
			}}
		}
	}
	return decisionengine.Accept, nil
}
