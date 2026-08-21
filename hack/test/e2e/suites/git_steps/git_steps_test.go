//go:build e2e
//nolint:forcetypeassert
package git_steps_test

// This suite exercises pull-request and tag git promotion steps with no Argo CD
// -- only git steps. Modeled on the prod stage of git_commit_only, it has one
// stage per step under test, each promoting freight directly from the Warehouse:
//
//   merge-pr    -- opens a PR and merges it with git-merge-pr (within the promotion)
//   wait-for-pr -- opens a PR and blocks on git-wait-for-pr until the test merges it
//   tag         -- creates and pushes an annotated tag with git-tag

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// Expected git-open-pr title/description/labels for the open-pr stage; kept in
// sync with testdata/kargo/kargo.yaml.
const (
	prTitle       = "git-steps e2e pull request"
	prDescription = "Opened by the git-open-pr e2e test."
)

var prLabels = []string{"kargo-e2e", "git-steps"}

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestGitSteps(t *testing.T) {
	feature := features.New("git-steps")

	project := "kargo-git-steps"
	origin := "kargo-demo"

	feature.Setup(utils.SetupKargoClients)

	// Substitute the git credentials Secret, the promotion's gitRepo var and the
	// Warehouse git subscription with the fork and PAT from the test env.
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		creds := utils.RequireGitCreds(ctx, t)
		return utils.NewSetupKargoFixtures(
			utils.UpdateGitCredentialsSecret("manifests", creds.RepoURL, creds.Username, creds.Password),
			utils.UpdateStagePromotionVar("", "gitRepo", creds.RepoURL),
			utils.UpdateWarehouseGitRepoURL("kargo-demo", creds.RepoURL),
		)(ctx, t, cfg)
	})
	feature.Teardown(utils.TeardownKargoFixtures)

	// Ensure the labels the open-pr stage applies exist on the fork; GitHub's
	// add-labels API requires labels to exist beforehand.
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		creds := utils.RequireGitCreds(ctx, t)
		for _, label := range prLabels {
			if err := utils.EnsureLabel(ctx, creds.RepoURL, creds.Password, label); err != nil {
				t.Fatalf("failed to ensure label %q: %v", label, err)
			}
		}
		return ctx
	})

	feature.Assess("require freight", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		t.Logf("Require freight \n")

		anyFreightID, err := utils.WaitForLatestFreight(ctx, project, origin, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Freight: %v", anyFreightID)
		return context.WithValue(ctx, envfuncs.ContextKey("freight_id"), anyFreightID)
	})

	feature.Assess("git-merge-pr merges the pull request", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		stage := "merge-pr"

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		// git-merge-pr merges the PR within the promotion, so a successful
		// promotion confirms the pull request was opened and merged.
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

	feature.Assess("git-wait-for-pr completes when the pull request is merged", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		creds := utils.RequireGitCreds(ctx, t)
		stage := "wait-for-pr"

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		// git-wait-for-pr blocks until the PR is merged; PromoteWithPRMerge reads
		// the PR number from the git-open-pr step and merges it via the API.
		if _, err := utils.PromoteWithPRMerge(
			ctx, t,
			project, stage, freightID,
			creds.RepoURL, creds.Password, "open-pr",
			15*time.Minute,
		); err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	feature.Assess("git-tag creates and pushes a tag", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		stage := "tag"

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		// A successful promotion confirms git-tag created the tag and git-push
		// published it.
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

	feature.Assess("git-open-pr sets title, description and labels", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		creds := utils.RequireGitCreds(ctx, t)
		stage := "open-pr"

		t.Logf("Promoting %v to %v \n", stage, freightID)
		if err := utils.RefreshStage(ctx, t, project, stage); err != nil {
			t.Fatal(err)
		}

		// The PR is left open (no merge step), so the promotion succeeds once the
		// PR is opened.
		promotion, err := utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, freightID,
			kargoapi.PromotionPhaseSucceeded,
			10*time.Minute,
		)
		if err != nil {
			t.Fatal(err)
		}

		prNumber, ok := utils.PullRequestID(promotion, "open-pr")
		if !ok {
			t.Fatalf("promotion did not record a pull request id; state: %v", promotion.Status.GetState())
		}

		pr, err := utils.GetPullRequest(ctx, creds.RepoURL, creds.Password, prNumber)
		if err != nil {
			t.Fatalf("error fetching pull request %d: %v", prNumber, err)
		}

		if pr.Title != prTitle {
			t.Fatalf("pull request title = %q, want %q", pr.Title, prTitle)
		}
		// git-open-pr may append a "View in Kargo UI" link to the body.
		if !strings.Contains(pr.Body, prDescription) {
			t.Fatalf("pull request body %q does not contain %q", pr.Body, prDescription)
		}
		for _, want := range prLabels {
			if !slices.Contains(pr.Labels, want) {
				t.Fatalf("pull request labels %v do not include %q", pr.Labels, want)
			}
		}

		t.Logf("pull request %d has expected title, description and labels", prNumber)
		return ctx
	})

	feature.Assess("git-clone checks out multiple individual commits", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		stage := "checkout-commits"

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

		// The final git-clone exposes each checked-out commit under its `as` key
		// in the `commits` output map; compare those to the created commits.
		outputs := map[string]string{}
		for _, key := range []string{"firstCheckout", "firstExpected", "secondCheckout", "secondExpected"} {
			val, ok := utils.PromotionStepOutput(promotion, "output", key)
			if !ok {
				t.Fatalf("promotion output is missing %q; state: %v", key, promotion.Status.GetState())
			}
			outputs[key] = val
		}

		if outputs["firstCheckout"] != outputs["firstExpected"] {
			t.Fatalf("first checkout %q does not match created commit %q", outputs["firstCheckout"], outputs["firstExpected"])
		}
		if outputs["secondCheckout"] != outputs["secondExpected"] {
			t.Fatalf("second checkout %q does not match created commit %q", outputs["secondCheckout"], outputs["secondExpected"])
		}
		if outputs["firstExpected"] == outputs["secondExpected"] {
			t.Fatalf("expected two distinct commits, both are %q", outputs["firstExpected"])
		}

		t.Logf("checked out commit %q as first and %q as second", outputs["firstCheckout"], outputs["secondCheckout"])
		return ctx
	})

	// FIXME: github-push integrates remote changes before pushing according to
	// the system-level push integration policy (AlwaysRebase, RebaseOrMerge,
	// RebaseOrFail, AlwaysMerge), which is configured via ClusterConfig / the
	// Helm chart -- not by step config. This test does not set up that policy, so
	// it only exercises the default (AlwaysRebase) with a non-diverged branch.
	// Setting up the push integration policy (and a diverged-branch scenario) to
	// verify the other policies is not yet implemented.
	feature.Assess("github-push pushes via the GitHub API", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		freightID := ctx.Value(envfuncs.ContextKey("freight_id")).(string)
		creds := utils.RequireGitCreds(ctx, t)
		stage := "github-push"

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

		branch, ok := utils.PromotionStepOutput(promotion, "output", "branch")
		if !ok {
			t.Fatalf("promotion output is missing branch; state: %v", promotion.Status.GetState())
		}
		commit, ok := utils.PromotionStepOutput(promotion, "output", "commit")
		if !ok {
			t.Fatalf("promotion output is missing commit; state: %v", promotion.Status.GetState())
		}

		// github-push updates the remote branch to point at the (replayed) commit
		// it reports, so the branch head on the remote must equal that commit.
		sha, err := utils.RemoteBranchSHA(ctx, creds.RepoURL, creds.Password, branch)
		if err != nil {
			t.Fatalf("error reading remote branch %q: %v", branch, err)
		}
		if sha != commit {
			t.Fatalf("remote branch %q head is %q, want the pushed commit %q", branch, sha, commit)
		}

		t.Logf("github-push pushed commit %q to branch %q", commit, branch)
		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())
}
