//go:build e2e
//nolint:forcetypeassert
package argocd_helm_commit_n_image

// This test implements an example of promoting argocd applications similar to https://github.com/akuity/kargo-examples
// The difference is that this example does not have an AnalysisTemplate verification.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdHelmCommitNImage(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
