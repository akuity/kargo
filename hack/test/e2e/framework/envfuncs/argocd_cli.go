package envfuncs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const ArgoCDConfigFile ContextKey = "argocd_config_file"

func LoadArgocdConfig(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	// TODO: other ways to discover/setup argocd config
	if argocdEnvConfig, err := GetEnv(ctx, []string{"argocd_cli", "config_file"}); err == nil {
		fileName := argocdEnvConfig.(string)
		if strings.HasPrefix(fileName, "~") {
			fileName = strings.Replace(fileName, "~", os.Getenv("HOME"), 1)
		}

		return context.WithValue(ctx, ArgoCDConfigFile, fileName), nil
	}
	fmt.Println("Cannot load argocd config from env")
	// Argocd config is optional. Do not fail here
	return ctx, nil
}
