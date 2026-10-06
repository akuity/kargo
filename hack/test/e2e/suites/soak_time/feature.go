//nolint:forcetypeassert
package soak_time

import (
	"context"
	"embed"
	"net/http"
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

// soakTime must match the requiredSoakTime configured on the uat Stage in
// testdata/kargo/kargo.yaml.
const soakTime = 2 * time.Minute

func feature() features.Feature {
	feature := features.New("soak-time")

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))

	project := "kargo-soak-time"
	origin := "kargo-demo"

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder.
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(utils.SetupKargoFixtures)
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

	feature.Assess("uat only accepts freight after soak time",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

			// Record the soak clock start before the freight enters test: the
			// freight becomes "currently in" test no earlier than this, so uat
			// cannot become eligible before soakStart + soakTime.
			soakStart := time.Now()

			t.Logf("Promoting test to %v \n", freightID)
			if err := utils.RefreshStage(ctx, t, project, "test"); err != nil {
				t.Fatal(err)
			}
			if _, err := utils.PromoteAndWaitForPhase(
				ctx, t,
				project, "test", freightID,
				kargoapi.PromotionPhaseSucceeded,
				10*time.Minute,
			); err != nil {
				t.Fatal(err)
			}
			utils.WaitForFreightToBeVerified(ctx, t, project, freightID, "test", 10*time.Minute)

			// The freight is now verified in test but has not soaked long enough,
			// so uat must reject a promotion attempt with 400 Bad Request.
			if err := utils.RefreshStage(ctx, t, project, "uat"); err != nil {
				t.Fatal(err)
			}
			status, err := utils.TryPromoteToStage(ctx, project, "uat", freightID)
			if err == nil || status != http.StatusBadRequest {
				t.Fatalf(
					"expected uat to reject freight before soak time with 400, got status %d, err %v",
					status, err)
			}
			if elapsed := time.Since(soakStart); elapsed >= soakTime {
				t.Fatalf(
					"soak time %v already elapsed (%v) before the rejection check; test is inconclusive",
					soakTime,
					elapsed)
			}
			t.Logf("uat correctly rejected freight before soak time (status %d)", status)

			// Promote to uat. StartPromotion retries the 400 until the freight has
			// soaked, so this call blocks until the soak time elapses and succeeds.
			t.Logf("Promoting uat (waiting out the %v soak time) \n", soakTime)
			if _, err := utils.PromoteAndWaitForPhase(
				ctx, t,
				project, "uat", freightID,
				kargoapi.PromotionPhaseSucceeded,
				soakTime+5*time.Minute,
			); err != nil {
				t.Fatal(err)
			}

			if elapsed := time.Since(soakStart); elapsed < soakTime {
				t.Fatalf("uat promotion succeeded after %v, before the required soak time of %v", elapsed, soakTime)
			} else {
				t.Logf("uat promotion succeeded after %v (>= soak time %v)", elapsed, soakTime)
			}

			utils.WaitForFreightToBeVerified(ctx, t, project, freightID, "uat", 10*time.Minute)

			return ctx
		})

	return feature.Feature()
}
