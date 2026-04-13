package health

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/updatecheck"
)

// UpdateCheck warns when a newer version is available.
type UpdateCheck struct {
	checker *updatecheck.Checker
}

// NewUpdateCheck creates an UpdateCheck using the given Checker.
func NewUpdateCheck(checker *updatecheck.Checker) *UpdateCheck {
	return &UpdateCheck{checker: checker}
}

func (c *UpdateCheck) Name() string { return "UpdateCheck" }

func (c *UpdateCheck) Check(ctx context.Context) []Result {
	result, err := c.checker.Check(ctx)
	if err != nil {
		return nil // don't report errors from update checking
	}
	if result.UpdateAvailable {
		return []Result{{
			Source:  "UpdateCheck",
			Type:    LevelNotice,
			Message: fmt.Sprintf("A newer version of sonarr2 is available: %s (current: %s)", result.LatestVersion, result.CurrentVersion),
		}}
	}
	return nil
}
