// Package health provides a framework for running health checks and
// aggregating results. Individual checks implement the Check interface.
package health

import (
	"context"
	"sync"
)

// Level indicates the severity of a health check result.
type Level string

const (
	LevelOK      Level = "ok"
	LevelNotice  Level = "notice"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

// Result is a single health check finding.
type Result struct {
	Source  string `json:"source"`
	Type    Level  `json:"type"`
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl,omitempty"`
}

// Check is a single health check that evaluates one aspect of system health.
type Check interface {
	Name() string
	Check(ctx context.Context) []Result
}

// Checker runs all registered health checks and aggregates results.
type Checker struct {
	checks []Check
	mu     sync.RWMutex
	last   []Result
}

// NewChecker creates a Checker with the given checks.
func NewChecker(checks ...Check) *Checker {
	return &Checker{checks: checks}
}

// RunAll runs every check and caches the aggregated results.
func (c *Checker) RunAll(ctx context.Context) []Result {
	var all []Result
	for _, ch := range c.checks {
		all = append(all, ch.Check(ctx)...)
	}
	if all == nil {
		all = []Result{}
	}
	c.mu.Lock()
	c.last = all
	c.mu.Unlock()
	return all
}

// Results returns the most recently cached results.
func (c *Checker) Results() []Result {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.last == nil {
		return []Result{}
	}
	return c.last
}
