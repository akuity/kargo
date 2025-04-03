package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/server/dex"
	"github.com/akuity/kargo/internal/server/oidc"
	"github.com/akuity/kargo/internal/types"
)

type StandardConfig struct {
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT" default:"30s"`
}

type ServerConfig struct {
	StandardConfig
	SecretManagementEnabled     bool
	LocalMode                   bool // LocalMode is true if the server is running as a non-containerized process
	TLSConfig                   *TLSConfig
	OIDCConfig                  *oidc.Config
	AdminConfig                 *AdminConfig
	DexProxyConfig              *dex.ProxyConfig
	ArgoCDConfig                ArgoCDConfig
	PermissiveCORSPolicyEnabled bool
	RolloutsIntegrationEnabled  bool
	AnalysisRunLogURLTemplate   string
	AnalysisRunLogToken         string
	AnalysisRunLogHTTPHeaders   map[string]string
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{}
	envconfig.MustProcess("", &cfg.StandardConfig)
	cfg.SecretManagementEnabled = types.MustParseBool(os.GetEnv("SECRET_MANAGEMENT_ENABLED", "false"))
	if types.MustParseBool(os.GetEnv("TLS_ENABLED", "false")) {
		tlsCfg := TLSConfigFromEnv()
		cfg.TLSConfig = &tlsCfg
	}
	if types.MustParseBool(os.GetEnv("OIDC_ENABLED", "false")) {
		oidcCfg := oidc.ConfigFromEnv()
		cfg.OIDCConfig = &oidcCfg
	}
	if types.MustParseBool(os.GetEnv("ADMIN_ACCOUNT_ENABLED", "false")) {
		adminCfg := AdminConfigFromEnv()
		cfg.AdminConfig = &adminCfg
	}
	if types.MustParseBool(os.GetEnv("DEX_ENABLED", "false")) {
		dexProxyCfg := dex.ProxyConfigFromEnv()
		cfg.DexProxyConfig = &dexProxyCfg
	}
	envconfig.MustProcess("", &cfg.ArgoCDConfig)
	cfg.PermissiveCORSPolicyEnabled =
		types.MustParseBool(os.GetEnv("PERMISSIVE_CORS_POLICY_ENABLED", "false"))
	cfg.RolloutsIntegrationEnabled =
		types.MustParseBool(os.GetEnv("ROLLOUTS_INTEGRATION_ENABLED", "true"))
	if cfg.RolloutsIntegrationEnabled {
		cfg.AnalysisRunLogURLTemplate = os.GetEnv("ANALYSIS_RUN_LOG_URL_TEMPLATE", "")
		cfg.AnalysisRunLogToken = os.GetEnv("ANALYSIS_RUN_LOG_TOKEN", "")
		if headersStr := os.GetEnv("ANALYSIS_RUN_LOG_HTTP_HEADERS", ""); headersStr != "" {
			kvPairs := strings.Split(headersStr, ",")
			cfg.AnalysisRunLogHTTPHeaders = make(map[string]string, len(kvPairs))
			for _, kvPair := range kvPairs {
				kv := strings.SplitN(kvPair, "=", 2)
				if len(kv) != 2 {
					panic(fmt.Sprintf("Invalid key-value pair: %s", kvPair))
				}
				cfg.AnalysisRunLogHTTPHeaders[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
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

// AdminConfig represents configuration for an admin account.
type AdminConfig struct {
	// HashedPassword is a bcrypt hash of the password for the admin account.
	HashedPassword string `envconfig:"ADMIN_ACCOUNT_PASSWORD_HASH" required:"true"`
	// TokenIssuer is the value to be used in the ISS claim of ID tokens issued for
	// the admin account.
	TokenIssuer string `envconfig:"ADMIN_ACCOUNT_TOKEN_ISSUER" required:"true"`
	// TokenAudience is the value to be used in the AUD claim of ID tokens issued
	// for the admin account.
	TokenAudience string `envconfig:"ADMIN_ACCOUNT_TOKEN_AUDIENCE" required:"true"`
	// TokenSigningKey is the key used to sign ID tokens for the admin account.
	TokenSigningKey []byte `envconfig:"ADMIN_ACCOUNT_TOKEN_SIGNING_KEY" required:"true"`
	// TokenTTL specifies how long ID tokens for the admin account are valid. i.e.
	// The expiry will be the time of issue plus this duration.
	TokenTTL time.Duration `envconfig:"ADMIN_ACCOUNT_TOKEN_TTL" default:"24h"`
}

// AdminConfigFromEnv returns an AdminConfig populated from environment
// variables.
func AdminConfigFromEnv() AdminConfig {
	var cfg AdminConfig
	envconfig.MustProcess("", &cfg)
	return cfg
}

type ArgoCDURLMap map[string]string

func (a *ArgoCDURLMap) Decode(value string) error {
	urls := make(map[string]string)
	if value != "" {
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			kvpair := strings.SplitN(pair, "=", 2)
			if len(kvpair) != 2 {
				return fmt.Errorf("invalid map item: %q. expected <shard>=<URL>", pair)
			}
			urls[strings.TrimSpace(kvpair[0])] = strings.TrimSpace(kvpair[1])

		}
	}
	*a = ArgoCDURLMap(urls)
	return nil
}

type ArgoCDConfig struct {
	// URLs is a mapping from shard name to Argo CD URL
	URLs ArgoCDURLMap `envconfig:"ARGOCD_URLS"`
}
