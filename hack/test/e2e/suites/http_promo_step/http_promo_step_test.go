//go:build e2e
//nolint:forcetypeassert
package http_promo_step_test

// This test implements the http promotion step example from
// https://github.com/akuity/kargo-examples (03-features/01-http-promo-step).
// The promotion posts a Slack-style message to an HTTP endpoint. The http step
// fails the promotion on a non-2xx response, so a successful promotion confirms
// the endpoint accepted the POST.
//
// The source example hard-codes the endpoint to the operator's host machine.
// This suite instead reads it from the test env (context.http_endpoint), which
// must point at an HTTP endpoint that accepts the POST and returns 2xx (e.g. an
// echo server). The multi-stage soak/verification pipeline is reduced to a
// single stage focused on the http step.

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestHTTPPromoStep(t *testing.T) {
	feature := features.New("http-promo-step")

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

	utils.TestEnv.Test(t, feature.Feature())
}
