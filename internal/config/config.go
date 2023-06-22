package config

import (
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/os"
)

type APIConfig struct {
	Host string
	Port int

	LocalMode bool

	GracefulShutdownTimeout time.Duration
}

func NewAPIConfig() APIConfig {
	return APIConfig{
		Host: os.MustGetEnv("HOST", "0.0.0.0"),
		Port: MustAtoi(os.MustGetEnv("PORT", "8080")),

		LocalMode: MustParseBool(os.MustGetEnv("LOCAL_MODE", "false")),

		GracefulShutdownTimeout: MustParseDuration(
			os.MustGetEnv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s"),
		),
	}
}

func (c APIConfig) RESTConfig() (*rest.Config, error) {
	return config.GetConfig()
}

type WebhooksConfig struct {
	ServiceAccount          string
	ServiceAccountNamespace string
}

func NewWebhooksConfig() WebhooksConfig {
	return WebhooksConfig{
		ServiceAccount: os.MustGetEnv(
			"KARGO_CONTROLLER_SERVICE_ACCOUNT",
			"",
		),
		ServiceAccountNamespace: os.MustGetEnv(
			"KARGO_CONTROLLER_SERVICE_ACCOUNT_NAMESPACE",
			"",
		),
	}
}
