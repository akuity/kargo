package envfuncs

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

const ArgoCDConfigFile ContextKey = "argocd_config_file"
const ArgocdHostKey ContextKey = "argocd_host"
const ArgocdPasswordKey ContextKey = "argocd_password"
const ArgocdUsernameKey ContextKey = "argocd_username"

func LoadArgocdConfig(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if argocdConfigFileVal := ctx.Value(ArgoCDConfigFile); argocdConfigFileVal != nil {
		// Config file already set, noop
		return ctx, nil
	}

	// TODO: other ways to discover/setup argocd config (such as using tempdir)
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


func ArgocdLogin(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	_, err := GetEnv(ctx, []string{"argocd_cli", "argocd_login"})
	if err != nil {
		fmt.Printf("Argocd login disabled, skipping \n")
		return ctx, nil
	}

	argocdHost, err := GetValueOrEnv(ctx, ArgocdHostKey, []string{"argocd_cli", "argocd_login", "argocd_host"})
	if err != nil {
		fmt.Printf("%v is not set, skipping argocd login \n", ArgocdHostKey)
		return ctx, nil
	}

	argocdPassword, err := GetValueOrEnv(ctx, ArgocdPasswordKey, []string{"argocd_cli", "argocd_login", "argocd_password"})
	if err != nil {
		fmt.Printf("%v is not set, skipping argocd login \n", ArgocdPasswordKey)
		return ctx, nil
	}

	argocdUsername, err := GetValueOrEnv(ctx, ArgocdUsernameKey, []string{"argocd_cli", "argocd_login", "argocd_username"})
	if err != nil {
		fmt.Printf("%v is not set, skipping argocd login \n", ArgocdUsernameKey)
		return ctx, nil
	}

	argocdConfigFile, ok := ctx.Value(ArgoCDConfigFile).(string)
	if !ok {
		fmt.Printf("%v is not set, skipping argocd login \n", ArgoCDConfigFile)
		return ctx, nil
	}

	fmt.Printf("Argocd login \n")

	cmd := fmt.Sprintf("argocd login --insecure %s --username %s --password %s --config %s", 
		argocdHost, argocdUsername, argocdPassword, argocdConfigFile)
	p := utils.RunCommandContext(ctx, cmd)
	if p.Err() != nil {
		outBytes, outErr := io.ReadAll(p.Out())
		if outErr != nil {
			return ctx, fmt.Errorf("argocd login failed: %w %w", p.Err(), outErr)
		}
		return ctx, fmt.Errorf("argocd login failed: %w : %s", p.Err(), outBytes)
	}

	return ctx, nil

}
