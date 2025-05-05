package external

import (
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
)

type StandardConfig struct {
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT" default:"30s"`
}

type ServerConfig struct {
	StandardConfig
	TLSConfig *TLSConfig
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{}
	envconfig.MustProcess("", &cfg.StandardConfig)
	if types.MustParseBool(os.GetEnv("TLS_ENABLED", "false")) {
		tlsCfg := TLSConfigFromEnv()
		cfg.TLSConfig = &tlsCfg
	}
	return cfg
}

type TLSConfig struct {
	CertPath string `envconfig:"TLS_CERT_PATH" required:"true"`
	KeyPath  string `envconfig:"TLS_KEY_PATH" required:"true"`
}

func TLSConfigFromEnv() TLSConfig {
	cfg := TLSConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}
