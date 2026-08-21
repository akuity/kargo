//go:build e2e
//nolint:forcetypeassert
package yaml_parse_update_test

// This test is adapted from the yaml-parse / yaml-update example at
// https://github.com/akuity/kargo-examples (03-features/03-yaml-parse-update).
//
// Differences from the source example:
//   - It uses the shared kargo-demo-gitops repository (like the other suites)
//     instead of the kargo-examples repository, parsing/updating image.name in
//     charts/kargo-demo/values.yaml on the new-helm branch.
//   - The git-commit and git-push steps are removed, so the working copy is
//     never written back to the repository.
//   - A second yaml-parse step re-reads the updated field, and the test asserts
//     that yaml-update actually changed it.

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

func TestYAMLParseUpdate(t *testing.T) {
	feature := features.New("yaml-parse-update")

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

	utils.TestEnv.Test(t, feature.Feature())
}
