package envfuncs

import (
	"context"
	"encoding/json"
	env "envs"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/utils"

	"github.com/akuity/kargo/pkg/cli/config"
	"sigs.k8s.io/yaml"
)

const KargoConfigKey ContextKey = "kargo_config"

const ConfigHomeVar string = "XDG_CONFIG_HOME"

func KargoLogin(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	loginConfigVal, err := GetEnv(ctx, []string{"kargo_cli", "kargo_login"})
	if err != nil {
		fmt.Printf("Kargo login disabled, skipping \n")
		return ctx, nil
	}

	kargoHost, err := GetValueOrEnv(ctx, KargoHostKey, []string{"kargo_cli", "kargo_login", "kargo_host"})
	if err != nil {
		fmt.Printf("%v is not set, skipping kargo login \n", KargoHostKey)
		return ctx, nil
	}

	kargoPassword, err := GetValueOrEnv(ctx, KargoPasswordKey, []string{"kargo_cli", "kargo_login", "kargo_password"})
	if err != nil {
		fmt.Printf("%v is not set, skipping kargo login \n", KargoPasswordKey)
		return ctx, nil
	}

	ctx, finalize, err := processLoginConfig(ctx, loginConfigVal)
	if finalize != nil {
		defer finalize()
	}
	if err != nil {
		return ctx, err
	}

	fmt.Printf("Kargo login \n")
	
	cmd := fmt.Sprintf("kargo login %s --admin --password %s", kargoHost, kargoPassword)
	p := utils.RunCommand(cmd)
	if p.Err() != nil {
		outBytes, outErr := io.ReadAll(p.Out())
		if outErr != nil {
			return ctx, fmt.Errorf("kargo login failed: %w %w", p.Err(), outErr)
		}
		return ctx, fmt.Errorf("kargo login failed: %w : %s", p.Err(), outBytes)
	}

	return ctx, nil
}

func processLoginConfig(ctx context.Context, loginConfigVal any) (context.Context, func(), error) {
	loginConfig, ok := loginConfigVal.(map[string]any)
	if ok {
		// There are extra fields in the config

		// Using tmpdir for config home
		if useTmpConfigHome, ok := loginConfig["use_tmp_config_home"]; ok {
			if useTmpConfigHome.(bool) {
				oldConfigHome := os.Getenv(ConfigHomeVar)

				tempdir := ctx.Value(TmpDirKey)
				if tempdir == nil {
					return ctx, nil, fmt.Errorf("Temp dir is not set up. Cannot create tmp confighome")
				}

				tmpConfigHome := filepath.Join(tempdir.(string), "config")

				configFile := filepath.Join(tmpConfigHome, "kargo", "config")
				ctx = context.WithValue(ctx, KargoConfigKey, configFile)

				os.Setenv(ConfigHomeVar, tmpConfigHome)
				return ctx, func() {
					if oldConfigHome == "" {
						os.Unsetenv(ConfigHomeVar)
					} else {
						os.Setenv(ConfigHomeVar, oldConfigHome)
					}
				}, nil
			}
		}
	}
	return ctx, nil, nil
}

func LoadKargoConfig(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if kargoConfig := ctx.Value(KargoConfigKey); kargoConfig != nil {
		// Config already set
		if fileName, ok := kargoConfig.(string); ok {
			// Config is set as a file
			kargoConfig, err := loadConfigFromFile(fileName)
			if err != nil {
				return ctx, err
			}
			return withKargoConfig(ctx, kargoConfig), nil
		}
		// Config is already parsed
		return ctx, nil
	}
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