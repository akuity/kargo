//go:build e2e
//nolint:forcetypeassert
package vars

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
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestVars(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
