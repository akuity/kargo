//go:build e2e
//nolint:forcetypeassert
package argocd_helm_chart_test

// This test implements the Argo CD driven, Helm chart-only example from
// https://github.com/akuity/kargo-examples (01-argocd-driven/02-helm-driven/01-chart-only).
// Stage-specific Argo CD Applications point at a specific version of the nginx
// chart in the Bitnami chart repository, and Kargo advances new chart versions
// from stage to stage. AnalysisTemplate verification is stripped (see
// testdata/review/verification.yaml).

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

func TestArgocdHelmChart(t *testing.T) {
	feature := features.New("argocd-helm-chart")

	project := "kargo-argocd-helm-chart"
	origin := "kargo-demo"

	feature.Setup(utils.SetupArgocdClient)
	feature.Setup(utils.SetupArgoCDFixtures)
	// feature.Teardown(utils.TeardownArgoCDFixtures)

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder.
	// This example subscribes to a public Helm chart repository, so no repo
	// URL substitution is required.
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

	for _, stage := range []string{"test", "uat", "prod"} {
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

	utils.TestEnv.Test(t, feature.Feature())
}
