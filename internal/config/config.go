package config

import (
	"time"

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
