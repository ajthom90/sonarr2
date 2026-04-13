package health

import (
	"context"
	"fmt"
)

// Pinger is the subset of db.Pool needed for health checking.
type Pinger interface {
	Ping(ctx context.Context) error
}

// DatabaseCheck verifies the database is reachable.
type DatabaseCheck struct {
	pool Pinger
}

func NewDatabaseCheck(pool Pinger) *DatabaseCheck {
	return &DatabaseCheck{pool: pool}
}

func (c *DatabaseCheck) Name() string { return "DatabaseCheck" }

func (c *DatabaseCheck) Check(ctx context.Context) []Result {
	if err := c.pool.Ping(ctx); err != nil {
		return []Result{{
			Source:  "DatabaseCheck",
			Type:    LevelError,
			Message: fmt.Sprintf("Unable to connect to database: %v", err),
		}}
	}
	return nil
}
