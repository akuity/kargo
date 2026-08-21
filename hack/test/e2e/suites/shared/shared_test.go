//go:build e2e && shared
//nolint:forcetypeassert
package shared_test

// This test implements the Argo CD driven, Helm chart-only example from
// https://github.com/akuity/kargo-examples (01-argocd-driven/02-helm-driven/01-chart-only).
// Stage-specific Argo CD Applications point at a specific version of the nginx
// chart in the Bitnami chart repository, and Kargo advances new chart versions
// from stage to stage. AnalysisTemplate verification is stripped (see
// testdata/review/verification.yaml).

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_fixtures"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart"
)

func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestAll(t *testing.T) {
	for _, feature := range utils.TestFeatures {
		utils.TestEnv.Test(t, feature)
	}
}
