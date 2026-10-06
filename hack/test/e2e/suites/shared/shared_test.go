//go:build e2e && shared

//nolint:forcetypeassert
package shared_test

// This suite runs all test features registered in utils.TestFeatures
// This is useful to share multiple suites in the same kind cluster (for performance)
// Suites need to be imported in order to run as a part of shared test

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_fixtures"

	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_chart_n_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_commit_n_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_image_chart_repo"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_helm_image_git_repo"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_kustomize_commit_n_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_kustomize_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_update"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/argocd_wait"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/git_commit_only"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/git_helm_commit_n_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/git_helm_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/git_kustomize_commit_n_image"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/git_kustomize_image"
	// _ "github.com/akuity/kargo/hack/test/e2e/suites/http_promo_step"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_fixtures"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/kargo_promotion_fail"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/soak_time"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/vars"
	_ "github.com/akuity/kargo/hack/test/e2e/suites/yaml_parse_update"
)

func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestAll(t *testing.T) {
	utils.TestEnv.TestInParallel(t, utils.TestFeatures...)
}
