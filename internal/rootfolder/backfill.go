package rootfolder

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ajthom90/sonarr2/internal/library"
)

// BackfillFromSeries inserts distinct filepath.Dir(series.path) values into
// root_folders. Idempotent — existing rows are skipped via ErrAlreadyExists.
// Call from app startup after db.Migrate succeeds.
func BackfillFromSeries(ctx context.Context, rf Store, series library.SeriesStore) error {
	all, err := series.List(ctx)
	if err != nil {
		return fmt.Errorf("rootfolder: backfill list series: %w", err)
	}
	seen := make(map[string]struct{})
	for _, s := range all {
		if s.Path == "" {
			continue
		}
		root := filepath.Dir(s.Path)
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		if _, err := rf.Create(ctx, root); err != nil && !errors.Is(err, ErrAlreadyExists) {
			return fmt.Errorf("rootfolder: backfill create %q: %w", root, err)
		}
	}
	return nil
}
