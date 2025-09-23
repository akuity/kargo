package main

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/kargo/pkg/cli/config"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadCLIConfig()
	if err != nil {
		if !config.IsConfigNotFoundErr(err) {
			_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("load config: %w", err))
			os.Exit(1)
		}
		cfg = config.NewDefaultCLIConfig()
	}
	cmd := NewRootCommand(cfg)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
