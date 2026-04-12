// Package specs contains the core decision engine specifications for sonarr2.
// Each spec implements the decisionengine.Spec interface and evaluates one
// specific criterion against a release and quality profile.
package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// QualityAllowedSpec rejects releases whose quality is not in the profile's
// allowed items list.
type QualityAllowedSpec struct{}

// Name implements decisionengine.Spec.
func (s QualityAllowedSpec) Name() string { return "QualityAllowed" }

// Evaluate implements decisionengine.Spec.
func (s QualityAllowedSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, profile profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	for _, it := range profile.Items {
		if it.QualityID == remote.QualityID {
			if it.Allowed {
				return decisionengine.Accept, nil
			}
			return decisionengine.Reject, []decisionengine.Rejection{
				{
					Type:   decisionengine.Permanent,
					Reason: fmt.Sprintf("quality %d is not allowed in profile", remote.QualityID),
					Spec:   s.Name(),
				},
			}
		}
	}
	// Quality not found in profile at all — treat as not allowed.
	return decisionengine.Reject, []decisionengine.Rejection{
		{
			Type:   decisionengine.Permanent,
			Reason: fmt.Sprintf("quality %d is not in profile", remote.QualityID),
			Spec:   s.Name(),
		},
	}
}
