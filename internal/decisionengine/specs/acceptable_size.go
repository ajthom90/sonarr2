package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

const mb = 1024 * 1024

// AcceptableSizeSpec rejects releases whose size falls outside the min/max
// range defined by the quality definition. Size zero means the indexer did not
// report a size — those are accepted (we cannot evaluate what we don't know).
type AcceptableSizeSpec struct {
	QualityDefs []profiles.QualityDefinition
}

// Name implements decisionengine.Spec.
func (s AcceptableSizeSpec) Name() string { return "AcceptableSize" }

// Evaluate implements decisionengine.Spec.
func (s AcceptableSizeSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if remote.Release.Size == 0 {
		// Unknown size — give benefit of the doubt.
		return decisionengine.Accept, nil
	}

	def, ok := s.defForQuality(remote.QualityID)
	if !ok {
		// No definition for this quality — cannot enforce size limits.
		return decisionengine.Accept, nil
	}

	minBytes := int64(def.MinSize * mb)
	if minBytes > 0 && remote.Release.Size < minBytes {
		return decisionengine.Reject, []decisionengine.Rejection{
			{
				Type:   decisionengine.Permanent,
				Reason: fmt.Sprintf("release size %d bytes is below minimum %d bytes for quality %s", remote.Release.Size, minBytes, def.Name),
				Spec:   s.Name(),
			},
		}
	}

	if def.MaxSize > 0 {
		maxBytes := int64(def.MaxSize * mb)
		if remote.Release.Size > maxBytes {
			return decisionengine.Reject, []decisionengine.Rejection{
				{
					Type:   decisionengine.Permanent,
					Reason: fmt.Sprintf("release size %d bytes exceeds maximum %d bytes for quality %s", remote.Release.Size, maxBytes, def.Name),
					Spec:   s.Name(),
				},
			}
		}
	}

	return decisionengine.Accept, nil
}

func (s AcceptableSizeSpec) defForQuality(qualityID int) (profiles.QualityDefinition, bool) {
	for _, d := range s.QualityDefs {
		if d.ID == qualityID {
			return d, true
		}
	}
	return profiles.QualityDefinition{}, false
}
