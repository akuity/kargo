//nolint:forcetypeassert
package argocd_helm_image_chart_repo

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
	feature := features.New("argocd-helm-image-chart-repo")

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))

	project := "kargo-argocd-helm-image-chartrepo"
	origin := "kargo-demo"

	feature.Setup(utils.SetupArgocdClient)
	feature.Setup(utils.SetupArgoCDFixtures)
	feature.Teardown(utils.TeardownArgoCDFixtures)

	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder.
	// This example subscribes to a public image registry and a public Helm
	// chart repository, so no repo URL substitution is required.
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

	return feature.Feature()
}
