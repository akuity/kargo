package main

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/kargo/pkg/cli/config"
)

func main() {
	ctx := context.Background()
	// Get config from env vars first.
	cfg := config.NewEnvVarCLIConfig()
	// If env vars provided insufficient connection details, try loading from
	// config file.
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		var err error
		if cfg, err = config.LoadCLIConfig(); err != nil {
			if !config.IsConfigNotFoundErr(err) {
				_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("load config: %w", err))
				os.Exit(1)
			}
			// If we get to here, we tried loading config from a file, but it wasn't
			// found. Fall back to defaults (which will be empty). Individual commands
			// will fail if they use a client that requires connection details.
			cfg = config.NewDefaultCLIConfig()
		}
	}
	cmd := NewRootCommand(cfg)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
