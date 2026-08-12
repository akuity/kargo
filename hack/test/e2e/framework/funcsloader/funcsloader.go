package funcsloader

import (
	"context"
	"slices"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func GetFuncs() ([]env.Func, []env.Func) {
	// Always load env file first
	baseSetup := []env.Func{
		// Always load env file first
		envfuncs.LoadEnvFile,
		envfuncs.SetupTempDir,
	}
	baseTeardown := []env.Func{
		envfuncs.TeardownTempDir,
	}

	configSetup := []env.Func{
		// We load config file AFTER login because login sets it up (different from argocd)
		envfuncs.KargoLogin,
		envfuncs.LoadKargoConfig,
		// We load config file BEFORE login because it's used as an argument (different from kargo)
		envfuncs.LoadArgocdConfig,
		envfuncs.ArgocdLogin,
	}

	// All setup functions should be added here
	clusterSetup := envfuncs.ClusterSetupFuncs()
	clusterTeardown := envfuncs.ClusterTeardownFuncs()

	return slices.Concat(baseSetup, clusterSetup, configSetup),
		slices.Concat(clusterTeardown, baseTeardown)

}

func noopFunc(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	return ctx, nil
}
