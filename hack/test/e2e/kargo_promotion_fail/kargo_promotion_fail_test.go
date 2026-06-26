//go:build e2e
//nolint:forcetypeassert
package kargo_promotion_fail_test

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/hack/test/e2e/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestKargoPromotionFail(t *testing.T) {
	feature := features.New("Example kargo promotion")
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

	utils.TestEnv.Test(t, feature.Feature())

}
