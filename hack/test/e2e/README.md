# E2E testing

This folder contains test suites to run e2e tests on live kargo instance.

Instance and other configurations can be configured in YAML files in the `envs` directory.
See example `dev.env` and `home_config.env`

Tests can be run as `go test <package_name>` with access to kargo instance set up in `envs`.

To run tests using your home-directory kargo config, aquired with `kargo login` command:
```
go test <package_name> -args -env-file=home_config.yaml
```

## Prerequisites

- `go` - tests are run with `go test`
- `kubectl`, `kargo` and `argocd` CLI tools installed - some tests and helper functions require those CLIs
- `kind` and `helm` CLI tools installed to run in local `kind` cluster

## Test framework used

This project uses the https://github.com/kubernetes-sigs/e2e-framework to set up environment and test runners.

See https://github.com/kubernetes-sigs/e2e-framework/blob/main/docs/design/README.md for more info.

This framework provides some useful tools, but not all of them are used at the moment. For example it provides tools to create a kind cluster and use kubeconfig, which are not used in the test examples

*The framework is used for convenience and is not strictly required for the test format, but replacing it will take some
work. This project mostly focuses on defining callbacks and helpers which can be built in the e2e-framework, or some other framework.*


## Writing tests

This module is using `sigs.k8s.io/e2e-framework` to configure and run tests.

We recommend organizing each feature into its own package.

Each test package should configure the following `TestMain`:
```
package my_package_test

import "github.com/akuity/kargo/hack/test/e2e/framework/utils"

func TestMain(m *testing.M) {
	utils.InitEnv(m)
}
```

### Test feature

Test functions should configure `e2e-framework` features:
```
func TestMyFeature(t *testing.T) {
	feature := features.New("Example feature")
	feature.Setup(...)
	feature.Teardown(...)
	feature.Assess(...)

	utils.TestEnv.Test(t, feature.Feature())
}
```

It's possible to create YAML fixtures for each test by placing them in the `testdata` folder in the test project and adding `SetupKargoFixtures` and `TeardownKargoFixtures` callbacks:
```
func TestMyFeature(t *testing.T) {
	feature := features.New("Example kargo promotion")
	
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(utils.SetupKargoFixtures)
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess(...)

	utils.TestEnv.Test(t, feature.Feature())
}
```

Please see `kargo_promotion_fail` as an example for running promotions in a project.

## Test environments

`envs` folder contains YAML files with configurations accessible to the tests, each `go test` run can be run with a particular environment. Environment configurations mainly used by the environment configuration functions in the `envfuncs` package.

## Provisioning a local cluster

By default the tests run against an already-running Kargo instance (see `home_config.yaml`). The `envfuncs` package can instead stand up a self-contained environment on a local [kind](https://kind.sigs.k8s.io/) cluster using the `sigs.k8s.io/e2e-framework` kind and Helm helpers:

- `CreateKindCluster` / `DestroyKindCluster` -- create and delete the cluster
- `InstallCertManager` -- installs cert-manager (a prerequisite for Kargo)
- `InstallArgoCD` -- installs the Argo CD Helm chart
- `InstallKargo` -- installs the Kargo Helm chart (optionally loading a locally built image into the cluster first)

`ClusterSetupFuncs()` and `ClusterTeardownFuncs()` return these as ordered `env.Func` slices. They are driven entirely by the env file: they run only when it contains a `cluster` section and otherwise no-op, so they are safe to add to a `funcsloader` unconditionally. See `envs/kind_config.yaml` for all configurable fields (cluster name/image, chart repos, versions, values files, `--set` overrides, and an optional Kargo image to load).

`prerequisites`: the `kind` and `helm` binaries must be available on `PATH`.

These env functions will be called for each test, but require env configuration with `cluster`, `cert_manager`, `argocd` and `kargo` keys to instruct the functions to actually set up envorinments.
**See envs/kind_config.yaml file for configuration reference**

## Dependency modules

This folder contains the main e2e test module `github.com/akuity/kargo/hack/test/e2e` and a few helper packages, each in its own module: `envs`, `envfuncs`, `funcsloader`.

The purpose of that is to be able to override `envs` and `funcsloader` in dependent `e2e` test modules to provide different environment configuration and setup/teardown functions to test in more environments than this package provides.

## TODO

- Allow disabling teardown on errors.
- Allow separate context values file from init congigurations
- Surface errors better
