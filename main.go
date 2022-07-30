package main

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/cmd/controller"
	"github.com/akuityio/k8sta/cmd/server"
	"github.com/akuityio/k8sta/internal/common/signals"
)

const binaryNameEnvVar = "K8STA_BINARY_NAME"

func main() {
	binaryName := filepath.Base(os.Args[0])
	if val := os.Getenv(binaryNameEnvVar); val != "" {
		binaryName = val
	}

	ctx := signals.Context()

	config, err := k8staConfig()
	if err != nil {
		log.Fatal(err)
	}

	switch binaryName {
	case "k8sta-controller":
		err = controller.RunController(ctx, config)
	case "k8sta-server":
		err = server.RunServer(ctx, config)
	default:
		err = errors.Errorf("unrecognized component name %q", binaryName)
	}

	if err != nil {
		log.Fatal(err)
	}
}
