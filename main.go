package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/akuityio/k8sta/cmd/bookkeeper"
	"github.com/akuityio/k8sta/cmd/cli"
	"github.com/akuityio/k8sta/cmd/controller"
	"github.com/akuityio/k8sta/cmd/server"
)

const binaryNameEnvVar = "K8STA_BINARY_NAME"

func main() {
	binaryName := filepath.Base(os.Args[0])
	if val := os.Getenv(binaryNameEnvVar); val != "" {
		binaryName = val
	}

	ctx := context.Background()

	config, err := k8staConfig()
	if err != nil {
		log.Fatal(err)
	}

	switch binaryName {
	case "bookkeeper",
		"bookkeeper-darwin-amd64", "bookkeeper-darwin-arm64",
		"bookkeeper-linux-amd64", "bookkeeper-linux-arm64",
		"bookkeeper.exe", "bookkeeper-windows-amd64.exe":
		var cmd *cobra.Command
		cmd, err = cli.NewRootCommand()
		if err != nil {
			log.Fatal(err)
		}
		if err = cmd.Execute(); err != nil {
			// Cobra will display the error for us. No need to do it ourselves.
			os.Exit(1)
		}
	case "bookkeeper-server":
		err = bookkeeper.RunServer(ctx, config)
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
