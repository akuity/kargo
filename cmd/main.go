package main

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/akuityio/kargo/internal/cmd"
	"github.com/akuityio/kargo/internal/logging"
)

func main() {
	ctx := signals.SetupSignalHandler()
	if err := cmd.Execute(ctx); err != nil {
		logging.LoggerFromContext(ctx).Error(err)
		os.Exit(1)
	}
}
