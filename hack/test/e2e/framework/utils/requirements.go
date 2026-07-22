package utils

import (
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
)

func RequireEnvValue(path []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		ctx = RequireContextValue(envfuncs.EnvKey)(ctx, t, cfg)
		_, err := envfuncs.GetEnv(ctx, path)
		if err != nil {
			t.Fatalf("cannot get value for path %v from context %v", path, ctx)
		}
		return ctx
	}
}

func RequireContextValue(key envfuncs.ContextKey) features.Func {
	return func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		val := ctx.Value(key)
		if val == nil {
			t.Fatalf("%v is required in context", key)
		}
		return ctx
	}
}

func RequireKargoCli(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return RequireContextValue("kargo_cli")(ctx, t, cfg)
}
