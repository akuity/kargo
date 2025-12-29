package main

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/akuity/kargo/pkg/logging"

	_ "github.com/gogo/protobuf/gogoproto"

	_ "time/tzdata"
)

func main() {
	ctx := signals.SetupSignalHandler()
	if err := Execute(ctx); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "")
		os.Exit(1)
	}
}
