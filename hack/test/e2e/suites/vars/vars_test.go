//go:build e2e
//nolint:forcetypeassert
package vars_test

// This test implements the vars example from
// https://github.com/akuity/kargo-examples (03-features/04-vars). It exercises
// promotion variables defined at four different levels, each used to look up a
// pokemon by name via pokeapi.co. Because pokeapi echoes the requested name
// back, the test can assert that every var resolved to the expected value:
//
//   pokemon_1 (pikachu)    -- Stage spec.vars
//   pokemon_2 (charmander) -- Stage promotionTemplate.spec.vars
//   pokemon_3 (bulbasaur)  -- Stage promotionTemplate.spec.steps[].vars
//   pokemon_4 (ditto)      -- PromotionTask spec.vars
//
// The fifth level -- a var in an AnalysisTemplate argument -- is covered by the
// "stage is verified" assessment, which requires the pokemon-query verification
// (parameterized by vars.pokemon_1) to succeed.

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

func TestVars(t *testing.T) {
	feature := features.New("vars")

	project := "kargo-vars"
	origin := "vars"
	stage := "vars"

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

	feature.Assess("vars resolve at every level", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
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
				t.Fatalf("promotion output is missing %q (%s); state: %v", e.key, e.level, promotion.Status.GetState())
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

	utils.TestEnv.Test(t, feature.Feature())
}
