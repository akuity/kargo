//go:build e2e
//nolint:forcetypeassert
package kargo_promotion_fail

// This test shows an example of running Kargo promotion with stage defined in YAML fixtures.
// Specifically it executes the `fail` stage and checks that promotion fails.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestKargoPromotionFail(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
