// SPDX-License-Identifier: GPL-3.0-or-later
// Blocklist rejection spec — permanently rejects releases whose source
// title is blocklisted for the series.

package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/blocklist"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// BlocklistedSpec consults the blocklist and rejects any release whose
// exact source title has been blocklisted for the series.
type BlocklistedSpec struct {
	Store blocklist.Store
}

// Name implements decisionengine.Spec.
func (s BlocklistedSpec) Name() string { return "Blocklisted" }

// Evaluate implements decisionengine.Spec. A nil Store is treated as
// "no blocklist configured" and always accepts.
func (s BlocklistedSpec) Evaluate(ctx context.Context, remote decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if s.Store == nil {
		return decisionengine.Accept, nil
	}
	entries, err := s.Store.ListBySeries(ctx, int(remote.SeriesID))
	if err != nil {
		// Failure to read the blocklist should not block grabs — accept and
		// rely on the store's own logging to surface the error.
		return decisionengine.Accept, nil
	}
	if blocklist.Matches(entries, int(remote.SeriesID), remote.Release.Title) {
		return decisionengine.Reject, []decisionengine.Rejection{{
			Type:   decisionengine.Permanent,
			Reason: fmt.Sprintf("Release %q is blocklisted for series %d", remote.Release.Title, remote.SeriesID),
			Spec:   s.Name(),
		}}
	}
	return decisionengine.Accept, nil
}
