package config

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/internal/os"
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

	GracefulShutdownTimeout time.Duration
}

func NewAPIConfig() APIConfig {
	return APIConfig{
		BaseConfig:              newBaseConfig(),
		Host:                    os.MustGetEnv("HOST", "0.0.0.0"),
		Port:                    MustAtoi(os.MustGetEnv("PORT", "50051")),
		GracefulShutdownTimeout: MustParseDuration(os.MustGetEnv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")),
	}
}

type APIProxyConfig struct {
	BaseConfig
	Host        string
	Port        int
	APIEndpoint string

	GracefulShutdownTimeout time.Duration
}

func NewAPIProxyConfig() APIProxyConfig {
	return APIProxyConfig{
		BaseConfig:              newBaseConfig(),
		Host:                    os.MustGetEnv("HOST", "0.0.0.0"),
		Port:                    MustAtoi(os.MustGetEnv("PORT", "8080")),
		APIEndpoint:             os.MustGetEnv("API_ENDPOINT", "localhost:50051"),
		GracefulShutdownTimeout: MustParseDuration(os.MustGetEnv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")),
	}
}

type ControllerConfig struct {
	BaseConfig
}

func NewControllerConfig() ControllerConfig {
	return ControllerConfig{
		BaseConfig: newBaseConfig(),
	}
}
