package main

import (
	"context"
	"os"

	"github.com/akuity/kargo/internal/cli/option"
)

func main() {
	var opt option.Option
	ctx := context.Background()
	if err := NewRootCommand(&opt).ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
