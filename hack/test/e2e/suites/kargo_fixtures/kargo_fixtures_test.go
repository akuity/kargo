//go:build e2e && examples

//nolint:forcetypeassert
package kargo_fixtures

// This test shows an example of using YAML files to define Kargo fixtures to use in tests.
// It sets up fixtures and verifies that they exist.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestKargoFixtures(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
