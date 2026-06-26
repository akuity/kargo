package envfuncs

import (
	"context"
	"encoding/json"
	"fmt"
	env "envs"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/akuity/kargo/pkg/cli/config"
	"sigs.k8s.io/yaml"
)

// func init() {
// 	// FIMXE: set up flags to configure kargo config here
// 	// flag.StringVar(&kargoAuthMethod, "kargo-auth-method", "", "")
// }

// func loadConfigFromFlags() config.CLIConfig {
// 	return config.CLIConfig{
// 		// AuthMethod: "admin", // FIXME: modify that? "kubeconfig" | "sso"
// 		// APIAddress: kargoApiAddress,
// 		// BearerToken: kargoBearerToken,
// 		// RefreshToken: kargoRefreshToken,
// 		// InsecureSkipTLSVerify: kargoInsecureSkipTLSVerify,
// 		// Project: kargoProject,
// 	}
// }

const KargoConfigKey ContextKey = "kargo_config"

func LoadKargoConfig(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if kargoEnv, err := GetEnvMap(ctx, []string{"kargo_cli"}); err == nil {
		if kargoConfigFile, ok := kargoEnv["config_file"].(string); ok {
			kargoConfig, err := loadConfigFromFile(kargoConfigFile)
			if err != nil {
				return ctx, err
			}
			return withKargoConfig(ctx, kargoConfig), nil
		}
		if kargoConfigEnv, ok := kargoEnv["kargo_config"].(map[string]any); ok {
			kargoConfig, err := loadConfigFromEnv(kargoConfigEnv)	
			if err != nil {
				return ctx, err
			}
			return withKargoConfig(ctx, kargoConfig), nil
		}
	}

	return ctx, fmt.Errorf("Cannot load kargo_config from env")
}

func withKargoConfig(ctx context.Context, kargoConfig config.CLIConfig) context.Context {
	return context.WithValue(ctx, KargoConfigKey, kargoConfig)
}

func loadConfigFromEnv(kargoEnv map[string]any) (cfg config.CLIConfig, err error) {
	kargoConfig := config.CLIConfig{}
	jsonData, err := json.Marshal(kargoEnv)
	if err != nil {
		return config.CLIConfig{}, err
	}
	err = json.Unmarshal(jsonData, &kargoConfig)
	return kargoConfig, err
}

func loadConfigFromFile(fileName string) (cfg config.CLIConfig, err error) {
	if strings.HasPrefix(fileName, "~") {
		fileName = strings.Replace(fileName, "~", os.Getenv("HOME"), 1)
	}
	var configBytes []byte
	if strings.HasPrefix(fileName, "/") {
		fmt.Printf("Reading kargo config from file %v\n", fileName)
		configBytes, err = os.ReadFile(fileName)
	} else {
		fmt.Printf("Reading kargo config from embedded env %v\n", fileName)
		configBytes, err = env.Envs.ReadFile(filepath.Join("envs", fileName))
	}
	
	if err != nil {
		return config.CLIConfig{}, err
	}
	if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
		return config.CLIConfig{}, err
	}
	return cfg, nil
}