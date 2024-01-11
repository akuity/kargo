package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadCLIConfig()
	if err != nil {
		if !config.IsConfigNotFoundErr(err) {
			fmt.Fprintln(os.Stderr, errors.Wrap(err, "load config"))
			os.Exit(1)
		}
		cfg = config.NewDefaultCLIConfig()
	}
	cmd, err := NewRootCommand(cfg, option.NewOption(cfg), &rootState{})
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "new root command"))
		os.Exit(1)
	}
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
