package config

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/os"
)

type BaseConfig struct {
	LogLevel logrus.Level
}

func newBaseConfig() BaseConfig {
	return BaseConfig{
		LogLevel: MustParseLogLevel(os.MustGetEnv("LOG_LEVEL", "INFO")),
	}
}

type APIConfig struct {
	BaseConfig
	Host string
	Port int

	LocalMode bool

	GracefulShutdownTimeout time.Duration
}

func NewAPIConfig() APIConfig {
	return APIConfig{
		BaseConfig: newBaseConfig(),
		Host:       os.MustGetEnv("HOST", "0.0.0.0"),
		Port:       MustAtoi(os.MustGetEnv("PORT", "8080")),
		LocalMode:  MustParseBool(os.MustGetEnv("LOCAL_MODE", "false")),
		GracefulShutdownTimeout: MustParseDuration(
			os.MustGetEnv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s"),
		),
	}
}

type ControllerConfig struct {
	BaseConfig
	ArgoCDNamespace         string
	ServiceAccount          string
	ServiceAccountNamespace string
}

func NewControllerConfig() ControllerConfig {
	return ControllerConfig{
		BaseConfig:      newBaseConfig(),
		ArgoCDNamespace: os.MustGetEnv("ARGOCD_NAMESPACE", "argocd"),
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
