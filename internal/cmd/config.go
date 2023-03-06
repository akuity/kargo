package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/os"
)

func kargoConfig() (config.Config, error) {
	config := config.Config{}
	var err error
	config.LogLevel, err = log.ParseLevel(os.GetEnvVar("LOG_LEVEL", "INFO"))
	return config, err
}
