//go:build e2e
//nolint:forcetypeassert
package argocd_wait

// This test exercises the argocd-wait promotion step. A branch is created in
// test setup (as a copy of the kustomize branch); the Argo CD Application tracks
// that branch with auto-sync enabled. The Kargo
// promotion checks out the existing branch, updates the image, commits and
// pushes, then uses argocd-wait to block until Argo CD reconciles the change
// (Healthy + Synced). Kargo never triggers the sync -- Argo CD's own auto-sync
// does -- which is exactly the scenario argocd-wait exists for.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdWait(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
