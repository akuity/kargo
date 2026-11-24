package envfuncs

import (
	"context"
	"flag"
	"fmt"
	"os"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/yaml"

	"github.com/akuity/kargo/hack/test/e2e/envs"
)

type ContextKey string

const EnvKey ContextKey = "env"
const TmpDirKey ContextKey = "tmpdir"

var envFileName string

func init() {
	flag.StringVar(&envFileName, "env-file", "home_config.yaml", "E2E test environment file")
}

func LoadEnvFile(ctx context.Context, _ *envconf.Config) (context.Context, error) {
	envMap := map[string]any{}
	configBytes, err := envs.Envs.ReadFile(envFileName)
	fmt.Printf("Env file: %s\n", envFileName)
	if err != nil {
		return ctx, err
	}
	if err := yaml.Unmarshal(configBytes, &envMap); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, EnvKey, envMap), nil
}

func GetEnv(ctx context.Context, path []string) (any, error) {
	env := ctx.Value(EnvKey)
	val := env
	for _, key := range path {
		if current, ok := val.(map[string]any); ok {
			if val, ok = current[key]; ok {
				continue
			}
			return nil, fmt.Errorf("path %v is not found in env map %v", path, env)
		}
		return nil, fmt.Errorf("env %v is not a map: %v", key, val)
	}
	return val, nil
}

func GetEnvMap(ctx context.Context, path []string) (map[string]any, error) {
	env, err := GetEnv(ctx, path)
	if err == nil {
		if envMap, ok := env.(map[string]any); ok {
			return envMap, nil
		}
		return nil, fmt.Errorf("cannot convert env to map %v", env)
	}
	return nil, err
}

func GetValueOrEnv(ctx context.Context, valueKey ContextKey, path []string) (any, error) {
	if value := ctx.Value(valueKey); value != nil {
		return value, nil
	}
	return GetEnv(ctx, path)
}

func SetupTempDir(ctx context.Context, _ *envconf.Config) (context.Context, error) {
	tempDir, err := os.MkdirTemp("", "kargo-e2e-")
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, TmpDirKey, tempDir), nil
}

func TeardownTempDir(ctx context.Context, _ *envconf.Config) (context.Context, error) {
	if tempDir, ok := ctx.Value(TmpDirKey).(string); ok {
		if err := os.RemoveAll(tempDir); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
