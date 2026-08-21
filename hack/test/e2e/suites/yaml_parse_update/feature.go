package yaml_parse_update

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
	feature := features.New("yaml-parse-update")

	feature.Setup(utils.TestData(TestData))

	project := "kargo-yaml-parse-update"
	origin := "kargo-demo"
	stage := "yaml-parse-update"

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder. The demo repo is public,
	// so no credentials are needed; only the repo URL is substituted with the
	// fork from the test env (Warehouse subscription and the promotion var).
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kargoDemoRepoVal, err := envfuncs.GetEnv(ctx, []string{"context", "kargo_demo_gitops_repo"})
		if err != nil {
			t.Fatalf("cannot get kargo_demo_gitops_repo %v", err)
		}
		kargoDemoRepo := kargoDemoRepoVal.(string)

		return utils.NewSetupKargoFixtures(
			utils.UpdateWarehouseGitRepoURL("kargo-demo", kargoDemoRepo),
			utils.UpdateStagePromotionVar("", "repoURL", kargoDemoRepo),
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

	feature.Assess("yaml-update updates the parsed field", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}
		promotion, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute,
		)
		if err != nil {
			t.Fatal(err)
		}

		original, ok := utils.PromotionStepOutput(promotion, "output", "originalImage")
		if !ok {
			t.Fatalf("promotion output %q is missing originalImage; state: %v", "output", promotion.Status.GetState())
		}
		updated, ok := utils.PromotionStepOutput(promotion, "output", "updatedImage")
		if !ok {
			t.Fatalf("promotion output %q is missing updatedImage; state: %v", "output", promotion.Status.GetState())
		}
		expected, ok := utils.PromotionStepOutput(promotion, "output", "expectedImage")
		if !ok {
			t.Fatalf("promotion output %q is missing expectedImage; state: %v", "output", promotion.Status.GetState())
		}

		// yaml-update must have written the new value, which the second
		// yaml-parse then read back.
		if updated != expected {
			t.Fatalf("yaml-update did not update the field: got image.name %q, want %q", updated, expected)
		}
		// The field must have actually changed from its original value.
		if updated == original {
			t.Fatalf("yaml-update left the field unchanged at %q", original)
		}

		t.Logf("yaml-update changed image.name from %q to %q", original, updated)
		return ctx
	})

	return feature.Feature()
}
