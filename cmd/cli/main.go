package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/cli/option"
)

func main() {
	ctx := context.Background()
	cmd, err := NewRootCommand(&option.Option{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errors.Wrap(err, "new root command"))
		os.Exit(1)
	}
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
