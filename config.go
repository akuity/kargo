package main

import (
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/common/os"
	log "github.com/sirupsen/logrus"
)

func k8staConfig() (config.Config, error) {
	config := config.Config{}
	var err error
	config.LogLevel, err = log.ParseLevel(os.GetEnvVar("LOG_LEVEL", "INFO"))
	return config, err
}
