package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/internal/common/config"
	"github.com/akuityio/kargo/internal/common/os"
)

func kargoConfig() (config.Config, error) {
	config := config.Config{}
	var err error
	config.LogLevel, err = log.ParseLevel(os.GetEnvVar("LOG_LEVEL", "INFO"))
	return config, err
}
