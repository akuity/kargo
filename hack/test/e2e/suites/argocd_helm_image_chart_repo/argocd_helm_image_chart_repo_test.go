//go:build e2e
//nolint:forcetypeassert
package argocd_helm_image_chart_repo

// This test implements the Argo CD driven, Helm image-only (with chart repo)
// example from https://github.com/akuity/kargo-examples
// (01-argocd-driven/02-helm-driven/02-image-only/01-with-chart-repo).
// Stage-specific Argo CD Applications point at a specific version of the
// chart in the chart repository and mix in specific versions of the image, which Kargo watches. 

import (
	"testing"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdHelmImageChartRepo(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
