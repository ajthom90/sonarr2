package health

import "context"

// MetadataSourceCheck warns if the TVDB API key is not configured.
type MetadataSourceCheck struct {
	apiKey string
}

func NewMetadataSourceCheck(apiKey string) *MetadataSourceCheck {
	return &MetadataSourceCheck{apiKey: apiKey}
}

func (c *MetadataSourceCheck) Name() string { return "MetadataSourceCheck" }

func (c *MetadataSourceCheck) Check(_ context.Context) []Result {
	if c.apiKey == "" {
		return []Result{{
			Source:  "MetadataSourceCheck",
			Type:    LevelWarning,
			Message: "TVDB API key is not configured. Series metadata refresh will not work",
		}}
	}
	return nil
}
