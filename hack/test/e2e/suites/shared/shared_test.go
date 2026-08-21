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

_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_image_git_repo"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_wait"
_ "github.com/akuity/kargo/hack/test/e2e/suites/git_kustomize_commit_n_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_promotion_fail"
_ "github.com/akuity/kargo/hack/test/e2e/suites/yaml_parse_update"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart_n_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_kustomize_commit_n_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/git_commit_only"
_ "github.com/akuity/kargo/hack/test/e2e/suites/git_kustomize_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_commit_n_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_kustomize_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/git_helm_commit_n_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/http_promo_step"
_ "github.com/akuity/kargo/hack/test/e2e/suites/soak_time"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_image_chart_repo"
_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_update"
_ "github.com/akuity/kargo/hack/test/e2e/suites/git_helm_image"
_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_fixtures"
_ "github.com/akuity/kargo/hack/test/e2e/suites/vars"
)

func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestAll(t *testing.T) {
	for _, feature := range utils.TestFeatures {
		utils.TestEnv.Test(t, feature)
	}
}
