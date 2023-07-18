package main

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/kargo/internal/cli/option"
)

func main() {
	var opt option.Option
	ctx := context.Background()
	cmd, err := NewRootCommand(&opt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
