// Command sonarr-migrate imports data from an existing Sonarr v3/v4 SQLite
// database into a fresh sonarr2 database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/migrate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	source := flag.String("source", "", "Path to source Sonarr SQLite database (required)")
	dest := flag.String("dest", "", "Path to destination sonarr2 SQLite database (required)")
	var remaps remapFlags
	flag.Var(&remaps, "remap", "Path remapping old:new (repeatable)")
	dryRun := flag.Bool("dry-run", false, "Validate without writing")
	skipHistory := flag.Bool("skip-history", false, "Skip history import")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	flag.Parse()

	if *source == "" || *dest == "" {
		fmt.Fprintln(os.Stderr, "Usage: sonarr-migrate --source <sonarr.db> --dest <sonarr2.db> [flags]")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		os.Exit(1)
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx := context.Background()

	// Open dest as sonarr2 SQLite pool, run migrations.
	destPool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         "file:" + *dest + "?_journal=WAL&_busy_timeout=5000",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("open destination database: %w", err)
	}
	defer destPool.Close()

	if err := db.Migrate(ctx, destPool); err != nil {
		return fmt.Errorf("migrate destination database: %w", err)
	}

	m, err := migrate.New(migrate.Options{
		SourcePath:  *source,
		DestPool:    destPool,
		Remaps:      remaps.parsed,
		DryRun:      *dryRun,
		SkipHistory: *skipHistory,
		Log:         log,
	})
	if err != nil {
		return fmt.Errorf("initialize migrator: %w", err)
	}
	defer m.Close()

	report, err := m.Run(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Migration complete:")
	fmt.Printf("  Series:           %d\n", report.Series)
	fmt.Printf("  Seasons:          %d\n", report.Seasons)
	fmt.Printf("  Episodes:         %d\n", report.Episodes)
	fmt.Printf("  Episode files:    %d\n", report.EpisodeFiles)
	fmt.Printf("  Quality profiles: %d\n", report.QualityProfiles)
	fmt.Printf("  Indexers:         %d\n", report.Indexers)
	fmt.Printf("  Download clients: %d\n", report.DownloadClients)
	fmt.Printf("  Notifications:    %d\n", report.Notifications)
	fmt.Printf("  History:          %d\n", report.History)
	if len(report.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(report.Warnings))
		for _, w := range report.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	return nil
}

// remapFlags implements flag.Value for repeatable --remap flags.
type remapFlags struct {
	parsed []migrate.PathRemap
}

func (f *remapFlags) String() string { return "" }

func (f *remapFlags) Set(val string) error {
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("remap must be old:new, got %q", val)
	}
	f.parsed = append(f.parsed, migrate.PathRemap{Old: parts[0], New: parts[1]})
	return nil
}
