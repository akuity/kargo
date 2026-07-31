//go:build e2e
//nolint:forcetypeassert
package git_helm_image_test

// This test implements the Git driven, Helm-based image-only example from
// https://github.com/akuity/kargo-examples
// (02-git-driven/02-helm-driven/01-image-only). Kargo watches a container
// image repository for new versions, renders the kargo-demo Helm chart with the
// updated image tag, and advances the result from stage to stage by committing
// the rendered manifests to a stage-specific branch, then pointing the Argo CD
// Application at the pushed commit.
//
// The prod stage opens a pull request and waits for it to be merged
// (git-open-pr / git-wait-for-pr); the test merges that PR with the configured
// PAT so the promotion can complete. AnalysisTemplate verification is stripped
// (see testdata/review/verification.yaml).

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

func TestGitHelmImage(t *testing.T) {
	feature := features.New("git-helm-image")

	project := "kargo-git-helm-image"
	origin := "kargo-demo"

	feature.Setup(utils.SetupArgocdClient)
	// Point the Argo CD ApplicationSet's source at the fork of the demo GitOps
	// repository, mirroring the substitution applied to the Kargo fixtures.
	feature.Setup(utils.SetupArgoCDFixturesWithRepoURL(project))
	feature.Teardown(utils.TeardownArgoCDFixtures)

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder. Substitute the git
	// credentials Secret and the per-Stage gitRepo var with the fork and PAT
	// from the test env. The Warehouse subscribes to an image, not git, so no
	// git repo URL substitution is applied to it.
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		creds := utils.RequireGitCreds(ctx, t)
		return utils.NewSetupKargoFixtures(
			utils.UpdateGitCredentialsSecret("manifests", creds.RepoURL, creds.Username, creds.Password),
			utils.UpdateStagePromotionVar("", "gitRepo", creds.RepoURL),
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

	// test and uat push directly to their stage branches; prod is handled
	// separately below because it is gated on a pull request merge.
	for _, stage := range []string{"test", "uat"} {
		feature.Assess("promote "+stage, func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

			t.Logf("Promoting %v to %v \n", stage, freightID)

			if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
				t.Fatal(err)
			}

			if _, err := utils.PromoteAndWaitForPhase(
				ctx, t,
				project, stage, freightID,
				kargoapi.PromotionPhaseSucceeded,
				10*time.Minute,
			); err != nil {
				t.Fatal(err)
			}

			_ = utils.WaitForFreightToBeVerified(ctx, t, project, freightID, stage, 10*time.Minute)

			return ctx
		})
	}

	feature.Assess("promote prod", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		creds := utils.RequireGitCreds(ctx, t)
		stage := "prod"

		t.Logf("Promoting prod (merging its pull request) to %v \n", freightID)

		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		if _, err := utils.PromoteWithPRMerge(
			ctx, t,
			project, stage, freightID,
			creds.RepoURL, creds.Password, "open-pr",
			15*time.Minute,
		); err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())
}
