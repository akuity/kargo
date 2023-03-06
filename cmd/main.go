package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
