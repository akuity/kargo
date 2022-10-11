package server

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/akuityio/k8sta/internal/common/file"
	"github.com/akuityio/k8sta/internal/common/http"
	libOS "github.com/akuityio/k8sta/internal/common/os"
	"github.com/akuityio/k8sta/internal/dockerhub"
)

// dockerhubFilterConfig populates configuration for the token filter used to
// authenticate webhooks from Docker Hub.
func dockerhubFilterConfig() (dockerhub.TokenFilterConfig, error) {
	config := dockerhub.NewTokenFilterConfig()
	tokensPath, err := libOS.GetRequiredEnvVar("DOCKERHUB_TOKENS_PATH")
	if err != nil {
		return config, err
	}
	var exists bool
	if exists, err = file.Exists(tokensPath); err != nil {
		return config, err
	}
	if !exists {
		return config, errors.Errorf("file %s does not exist", tokensPath)
	}
	tokenBytes, err := os.ReadFile(tokensPath)
	if err != nil {
		return config, err
	}
	tokens := map[string]string{}
	if err :=
		json.Unmarshal(tokenBytes, &tokens); err != nil {
		return config, err
	}
	for _, token := range tokens {
		config.AddToken(token)
	}
	return config, nil
}

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
