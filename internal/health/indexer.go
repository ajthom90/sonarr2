package health

import "context"

// EnabledCounter counts the number of enabled provider instances.
type EnabledCounter interface {
	CountEnabled(ctx context.Context) (int, error)
}

// IndexerCheck warns if no indexers are configured.
type IndexerCheck struct {
	counter EnabledCounter
}

func NewIndexerCheck(counter EnabledCounter) *IndexerCheck {
	return &IndexerCheck{counter: counter}
}

func (c *IndexerCheck) Name() string { return "IndexerCheck" }

func (c *IndexerCheck) Check(ctx context.Context) []Result {
	n, err := c.counter.CountEnabled(ctx)
	if err != nil {
		return []Result{{
			Source:  "IndexerCheck",
			Type:    LevelError,
			Message: "Failed to check indexer configuration",
		}}
	}
	if n == 0 {
		return []Result{{
			Source:  "IndexerCheck",
			Type:    LevelWarning,
			Message: "No indexers are enabled. Sonarr will not be able to find new releases automatically",
		}}
	}
	return nil
}
