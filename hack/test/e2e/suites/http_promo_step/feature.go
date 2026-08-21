package http_promo_step

import (
	"context"
	"embed"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

func init() {
	utils.TestFeatures = append(utils.TestFeatures, feature())
}

var (
	//go:embed testdata/*
	TestData embed.FS
)

func feature() features.Feature {
	feature := features.New("http-promo-step")

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))

	project := "kargo-http-promo-step"
	origin := "kargo-demo"
	stage := "test"

	// Skip the tests if http_endpoint is not set
	feature.Setup(utils.SkipIfNoEnvValue([]string{"context", "http_endpoint"}))

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder. Substitute the http
	// endpoint the promotion posts to with the one configured in the test env.
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		endpointVal, err := envfuncs.GetEnv(ctx, []string{"context", "http_endpoint"})
		if err != nil {
			t.Fatalf("cannot get context.http_endpoint from env; configure it to an HTTP endpoint that returns 2xx to a POST: %v", err)
		}
		endpoint := endpointVal.(string)

		return utils.NewSetupKargoFixtures(
			utils.UpdatePromotionTasksVar("promo-process", "url", endpoint),
		)(ctx, t, cfg)
	})
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("require freight", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		t.Logf("Require freight \n")

		anyFreightID, err := utils.WaitForLatestFreight(ctx, project, origin, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Freight: %v", anyFreightID)
		return context.WithValue(ctx, envfuncs.ContextKey("freight_id"), anyFreightID)
	})

	feature.Assess("http step posts to the configured endpoint", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		// The http step fails the promotion on a non-2xx response, so reaching
		// Succeeded means the configured endpoint accepted the POST.
		if _, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute,
		); err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	return feature.Feature()
}
