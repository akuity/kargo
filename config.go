package main

import (
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/common/os"
	log "github.com/sirupsen/logrus"
)

func k8staConfig() (config.Config, error) {
	config := config.Config{}
	var err error
	if config.LogLevel, err =
		log.ParseLevel(os.GetEnvVar("LOG_LEVEL", "INFO")); err != nil {
		return config, err
	}
	if config.ArgoCDNamespace, err =
		os.GetRequiredEnvVar("ARGOCD_NAMESPACE"); err != nil {
		return config, err
	}
	if config.K8sTANamespace, err =
		os.GetRequiredEnvVar("K8STA_NAMESPACE"); err != nil {
		return config, err
	}
	return config, nil
}
