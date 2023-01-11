package server

import (
	"github.com/akuityio/k8sta/internal/common/http"
	libOS "github.com/akuityio/k8sta/internal/common/os"
)

// serverConfig populates configuration for the HTTP/S server from environment
// variables.
func serverConfig() (http.ServerConfig, error) {
	config := http.ServerConfig{}
	var err error
	config.Port, err = libOS.GetIntFromEnvVar("PORT", 8080)
	if err != nil {
		return config, err
	}
	config.TLSEnabled, err = libOS.GetBoolFromEnvVar("TLS_ENABLED", false)
	if err != nil {
		return config, err
	}
	if config.TLSEnabled {
		config.TLSCertPath, err = libOS.GetRequiredEnvVar("TLS_CERT_PATH")
		if err != nil {
			return config, err
		}
		config.TLSKeyPath, err = libOS.GetRequiredEnvVar("TLS_KEY_PATH")
		if err != nil {
			return config, err
		}
	}
	return config, nil
}
