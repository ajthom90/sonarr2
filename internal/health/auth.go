package health

import "context"

// AuthCheck warns about weak authentication configuration.
type AuthCheck struct {
	authMode string
}

// NewAuthCheck creates an AuthCheck.
func NewAuthCheck(authMode string) *AuthCheck {
	return &AuthCheck{authMode: authMode}
}

func (c *AuthCheck) Name() string { return "AuthenticationCheck" }

func (c *AuthCheck) Check(_ context.Context) []Result {
	if c.authMode == "none" {
		return []Result{{
			Source:  "AuthenticationCheck",
			Type:    LevelWarning,
			Message: "Authentication is disabled. Anyone with access to the network can access sonarr2",
		}}
	}
	return nil
}
