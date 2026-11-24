//go:build e2e
//nolint:forcetypeassert
package argocd_helm_chart_n_image

// This test implements the Argo CD driven, Helm chart-and-image example from
// https://github.com/akuity/kargo-examples (01-argocd-driven/02-helm-driven/03-chart-n-image).
// Stage-specific Argo CD Applications point at a specific version of the
// chart in the chart repository and set the image tag from a
// public image repository, and Kargo advances new chart versions and image
// tags from stage to stage.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdHelmChartNImage(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
