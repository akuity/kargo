//go:build e2e
//nolint:forcetypeassert
package soak_time

// This test implements the soak-time example from
// https://github.com/akuity/kargo-examples (03-features/02-soak-time). Freight
// must "soak" in an upstream stage for a required duration before it becomes
// eligible for promotion to the downstream stage. The example uses a 10m soak;
// this suite reduces it to 2m so the test can exercise the behavior end to end.
//
// The test promotes freight to test, then asserts that uat rejects the freight
// until the soak time has elapsed and only accepts it afterwards.
// AnalysisTemplate verification is stripped (see testdata/review/verification.yaml).

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestSoakTime(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
