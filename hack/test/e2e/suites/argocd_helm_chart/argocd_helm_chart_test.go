//go:build e2e
//nolint:forcetypeassert
package argocd_helm_chart

// This test implements the Argo CD driven, Helm chart-only example from
// https://github.com/akuity/kargo-examples (01-argocd-driven/02-helm-driven/01-chart-only).
// Stage-specific Argo CD Applications point at a specific version of the nginx
// chart in the Bitnami chart repository, and Kargo advances new chart versions
// from stage to stage. AnalysisTemplate verification is stripped (see
// testdata/review/verification.yaml).

import (
	"testing"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdHelmChart(t *testing.T) {
	// Actual test code lives in test_code.go
	// This is a trick to allow shared run between multiple packages
	utils.TestEnv.Test(t, feature())
}
