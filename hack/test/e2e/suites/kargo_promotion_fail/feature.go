//nolint:forcetypeassert
package kargo_promotion_fail

import (
	"context"
	"embed"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	feature := features.New("Example kargo promotion")
	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))
	project := "kargo-promotion-fail"
	// Setup and teardown fixtures from testdata folder
	feature.Setup(utils.SetupKargoClients)
	feature.Setup(utils.SetupKargoFixtures)
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("promotion fails", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		stage := "kargo-promotion-fail-stage"
		origin := "images"

		anyFreightID, err := utils.WaitForLatestFreight(ctx, project, origin, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_, err = utils.PromoteAndWaitForPhase(
			ctx, t,
			project, stage, anyFreightID,
			kargoapi.PromotionPhaseFailed,
			5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	return feature.Feature()
}
