//nolint:forcetypeassert
package vars

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
	feature := features.New("vars")

	project := "kargo-vars"
	origin := "vars"
	stage := "vars"

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))
	feature.Setup(utils.SetupKargoClients)

	// The chart Warehouse and pokeapi are public, so no substitution is needed.
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

	feature.Assess("vars resolve at every level",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
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

			// key -> (var level, expected pokemon name)
			expected := []struct {
				key   string
				level string
				value string
			}{
				{"pokemon1", "Stage spec.vars", "pikachu"},
				{"pokemon2", "Stage promotionTemplate.spec.vars", "charmander"},
				{"pokemon3", "Stage promotionTemplate.spec.steps[].vars", "bulbasaur"},
				{"pokemon4", "PromotionTask spec.vars", "ditto"},
			}
			for _, e := range expected {
				got, ok := utils.PromotionStepOutput(promotion, "output", e.key)
				if !ok {
					t.Fatalf(
						"promotion output is missing %q (%s); state: %v",
						e.key, e.level, promotion.Status.GetState())
				}
				if got != e.value {
					t.Fatalf("var at %s resolved to %q, want %q", e.level, got, e.value)
				}
				t.Logf("var at %s resolved to %q", e.level, got)
			}

			return ctx
		})

	feature.Assess("stage is verified", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
		// The AnalysisTemplate argument is populated from vars.pokemon_1, the
		// fifth var level; a successful verification confirms it resolved.
		utils.WaitForStageVerified(ctx, t, project, stage, 10*time.Minute)
		t.Logf("stage %q verified successfully", stage)
		return ctx
	})

	return feature.Feature()
}
