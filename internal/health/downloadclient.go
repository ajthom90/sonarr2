package health

import "context"

// DownloadClientCheck warns if no download clients are configured.
type DownloadClientCheck struct {
	counter EnabledCounter
}

func NewDownloadClientCheck(counter EnabledCounter) *DownloadClientCheck {
	return &DownloadClientCheck{counter: counter}
}

func (c *DownloadClientCheck) Name() string { return "DownloadClientCheck" }

func (c *DownloadClientCheck) Check(ctx context.Context) []Result {
	n, err := c.counter.CountEnabled(ctx)
	if err != nil {
		return []Result{{
			Source:  "DownloadClientCheck",
			Type:    LevelError,
			Message: "Failed to check download client configuration",
		}}
	}
	if n == 0 {
		return []Result{{
			Source:  "DownloadClientCheck",
			Type:    LevelWarning,
			Message: "No download clients are enabled. Sonarr will not be able to download releases",
		}}
	}
	return nil
}
