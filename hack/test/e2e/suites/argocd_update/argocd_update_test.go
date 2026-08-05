//go:build e2e
//nolint:forcetypeassert
package argocd_update_test

// This test implements an example of promoting argocd applications similar to https://github.com/akuity/kargo-examples
// The difference is that this example does not have an AnalysisTemplate verification.

import (
	"context"
	"testing"
	"time"

	// "github.com/akuity/kargo/pkg/x/client/generated/core"
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

func TestArgocdUpdate(t *testing.T) {
	feature := features.New("argocd-update")

	project := "kargo-argocd-update"

	feature.Setup(utils.SetupArgocdClient)
	feature.Setup(utils.SetupArgoCDFixtures)
	feature.Teardown(utils.TeardownArgoCDFixtures)

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kargoDemoRepoVal, err := envfuncs.GetEnv(ctx, []string{"context", "kargo_demo_gitops_repo"})
		if err != nil {
			t.Fatalf("cannot get kargo_demo_gitops_repo %v", err)
		}
		kargoDemoRepo := kargoDemoRepoVal.(string)

		return utils.NewSetupKargoFixtures(
			utils.UpdatePromotionTasksVar("promo-process", "gitRepo", kargoDemoRepo),
			utils.UpdateWarehouseGitRepoURL("kargo-demo", kargoDemoRepo),
		)(ctx, t, cfg)
	})
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("require freight", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		origin := "kargo-demo"

		t.Logf("Require freight \n")

		anyFreightId, err := utils.WaitForLatestFreight(ctx, project, origin, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Freight: %v", anyFreightId)
		return context.WithValue(ctx, envfuncs.ContextKey("freight_id"), anyFreightId)
	})

	feature.Assess("promote test", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		stage := "test"

		t.Logf("Promoting test to %v \n", freightID)

		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		_, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	feature.Assess("promote uat", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		stage := "uat"

		t.Logf("Promoting uat \n")

		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		_, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	feature.Assess("promote prod", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

		stage := "prod"

		t.Logf("Promoting prod \n")

		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		_, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())
}
