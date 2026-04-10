// Command sonarr is the sonarr2 server binary.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ajthom90/sonarr2/internal/app"
	"github.com/ajthom90/sonarr2/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "sonarr2: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	a := app.New(cfg)
	ctx, cancel := app.SignalContext(context.Background())
	defer cancel()
	return a.Run(ctx)
}
