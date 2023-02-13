package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/cmd/controller"
	"github.com/akuityio/kargo/internal/version"
)

const binaryNameEnvVar = "KARGO_BINARY_NAME"

func main() {
	binaryName := filepath.Base(os.Args[0])
	if val := os.Getenv(binaryNameEnvVar); val != "" {
		binaryName = val
	}

	if len(os.Args) > 1 && os.Args[1] == "version" {
		versionBytes, err := json.MarshalIndent(version.GetVersion(), "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(versionBytes))
		return
	}

	ctx := context.Background()

	config, err := kargoConfig()
	if err != nil {
		log.Fatal(err)
	}

	switch binaryName {
	case "kargo-controller":
		err = controller.RunController(ctx, config)
	default:
		err = errors.Errorf("unrecognized component name %q", binaryName)
	}

	if err != nil {
		log.Fatal(err)
	}
}
