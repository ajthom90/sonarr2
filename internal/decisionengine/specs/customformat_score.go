package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// CustomFormatScoreSpec rejects releases whose custom format score is below
// the profile's MinFormatScore threshold.
type CustomFormatScoreSpec struct{}

// Name implements decisionengine.Spec.
func (s CustomFormatScoreSpec) Name() string { return "CustomFormatScore" }

// Evaluate implements decisionengine.Spec.
func (s CustomFormatScoreSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, profile profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if remote.CFScore >= profile.MinFormatScore {
		return decisionengine.Accept, nil
	}
	return decisionengine.Reject, []decisionengine.Rejection{
		{
			Type:   decisionengine.Permanent,
			Reason: fmt.Sprintf("custom format score %d is below minimum %d", remote.CFScore, profile.MinFormatScore),
			Spec:   s.Name(),
		},
	}
}
