//nolint:forcetypeassert
package argocd_wait

import (
	"context"
	"embed"
	"fmt"
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
	feature := features.New("argocd-wait")

	project := "kargo-argocd-wait"
	origin := "kargo-demo"
	stage := "test"

	// Branch created uniquely per test run. Both the Argo CD Application's
	// targetRevision and the promotion's targetBranch var point at it.
	branch := fmt.Sprintf("argocd-wait/e2e/%d", time.Now().UnixNano())

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))

	// Create the branch (a copy of the kustomize branch) BEFORE Argo CD is set
	// up, so the Application has an existing branch to track.
	feature.Setup(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		creds := utils.RequireGitCreds(ctx, t)
		t.Logf("Git creds %v", creds)

		if err := utils.CreateRemoteBranch(ctx, creds.RepoURL, creds.Password, branch, "kustomize"); err != nil {
			t.Fatalf("failed to create branch %q: %v", branch, err)
		}
		t.Logf("created branch %q", branch)
		return ctx
	})

	feature.Setup(utils.SetupArgocdClient)
	// Point the Argo CD ApplicationSet at the fork and the per-run branch.
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		repoVal, err := envfuncs.GetEnv(ctx, []string{"context", "kargo_demo_gitops_repo"})
		if err != nil {
			t.Fatalf("cannot get kargo_demo_gitops_repo %v", err)
		}
		repo := repoVal.(string)

		return utils.NewSetupArgoCDFixtures(
			utils.UpdateApplicationSetRepoURL(project, repo),
			utils.UpdateApplicationSetTargetRevision(project, branch),
		)(ctx, t, cfg)
	})
	feature.Teardown(utils.TeardownArgoCDFixtures)

	feature.Setup(utils.SetupKargoClients)

	// Substitute the git credentials Secret, the promotion's gitRepo var and the
	// per-run targetBranch. The Warehouse subscribes to an image, so no Warehouse
	// git repo URL substitution is applied.
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		creds := utils.RequireGitCreds(ctx, t)
		return utils.NewSetupKargoFixtures(
			utils.UpdateGitCredentialsSecret("manifests", creds.RepoURL, creds.Username, creds.Password),
			utils.UpdateStagePromotionVar("", "gitRepo", creds.RepoURL),
			utils.UpdateStagePromotionVar("", "targetBranch", branch),
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

	feature.Assess("promotion waits for argocd to reconcile",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)

			t.Logf("Promoting %v to %v on branch %q \n", stage, freightID, branch)
			if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
				t.Fatal(err)
			}

			// The argocd-wait step keeps the promotion Running until Argo CD reports
			// the Application Healthy and Synced, so reaching Succeeded means the
			// pushed change was reconciled.
			if _, err := utils.PromoteAndWaitForPhase(
				ctx, t,
				project, stage, freightID,
				kargoapi.PromotionPhaseSucceeded,
				15*time.Minute,
			); err != nil {
				t.Fatal(err)
			}

			return ctx
		})

	return feature.Feature()
}
