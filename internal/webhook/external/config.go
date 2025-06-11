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
	TLSConfig               *TLSConfig
	BaseURL                 string
	ClusterSecretsNamespace string
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{
		BaseURL:                 os.GetEnv("EXTERNAL_WEBHOOK_SERVER_BASE_URL", ""),
		ClusterSecretsNamespace: os.GetEnv("CLUSTER_SECRETS_NAMESPACE", ""),
	}
	if cfg.BaseURL == "" {
		panic("EXTERNAL_WEBHOOK_SERVER_BASE_URL must be set")
	}
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
