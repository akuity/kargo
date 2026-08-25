 // This package defines the set of environment configuration functions.
 // It is a separate package to allow replacing it without changing the main `InitEnv`
 // This allows calling test for OSS packages from EE codebase.
 package funcsloader

import (
	"slices"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

// GetFuncs provides an ordered list Setup and Teardown functions for test env (see scaffolding.go)
// All functions will be added to the environment and called when test is run,
// however function definitions may decide to return early and not perform a setup 
// based on configuration.
func GetFuncs() ([]env.Func, []env.Func) {
	

	baseSetup := []env.Func{
		// Load a yaml file set by `--env-file` and makes it accessible via context.
		envfuncs.LoadEnvFile,
		// Create a temporary directory for the test run
		// e2e-framework doesn't make it easy to access the tests tempdir so we set one here
		envfuncs.SetupTempDir,
	}
	baseTeardown := []env.Func{
		envfuncs.TeardownTempDir,
	}

	// Load kargo and argocd config. Optionally log in. See envfuncs/kargo_cli.go and envfuncs/argocd_cli.go
	// Functions may skip setup based on configuration from env file.
	configSetup := []env.Func{
		// We load config file AFTER login because login sets it up (different from argocd)
		envfuncs.KargoLogin,
		envfuncs.LoadKargoConfig,
		// We load config file BEFORE login because it's used as an argument (different from kargo)
		envfuncs.LoadArgocdConfig,
		envfuncs.ArgocdLogin,
	}

	// Functions setting up kind cluster and installing kargo and argocd as helm charts. See envfuncs/cluster.go
	// Functions may skip setup based on configuration from env file.
	clusterSetup := envfuncs.ClusterSetupFuncs()
	clusterTeardown := envfuncs.ClusterTeardownFuncs()

	return slices.Concat(baseSetup, clusterSetup, configSetup),
		slices.Concat(clusterTeardown, baseTeardown)

}
