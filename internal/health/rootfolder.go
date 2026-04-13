package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// SeriesPathLister lists series paths for root folder checking.
type SeriesPathLister interface {
	ListRootPaths(ctx context.Context) ([]string, error)
}

// RootFolderCheck verifies that series root folders exist on disk.
type RootFolderCheck struct {
	lister SeriesPathLister
}

func NewRootFolderCheck(lister SeriesPathLister) *RootFolderCheck {
	return &RootFolderCheck{lister: lister}
}

func (c *RootFolderCheck) Name() string { return "RootFolderCheck" }

func (c *RootFolderCheck) Check(ctx context.Context) []Result {
	paths, err := c.lister.ListRootPaths(ctx)
	if err != nil {
		return []Result{{
			Source:  "RootFolderCheck",
			Type:    LevelError,
			Message: fmt.Sprintf("Failed to list root paths: %v", err),
		}}
	}
	var results []Result
	seen := map[string]bool{}
	for _, p := range paths {
		root := filepath.Dir(p)
		if seen[root] {
			continue
		}
		seen[root] = true
		if _, err := os.Stat(root); os.IsNotExist(err) {
			results = append(results, Result{
				Source:  "RootFolderCheck",
				Type:    LevelWarning,
				Message: fmt.Sprintf("Root folder is missing: %s", root),
			})
		}
	}
	return results
}
