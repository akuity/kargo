package funcsloader

import (
	"context"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
)

func GetFuncs() ([]env.Func, []env.Func) {
	// All setup functions should be added here
	return []env.Func{
			envfuncs.LoadEnvFile,
			envfuncs.LoadKargoConfig,
			envfuncs.LoadArgocdConfig,
		},
		[]env.Func{
			noopFunc,
		}

}

func noopFunc(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	return ctx, nil
}
