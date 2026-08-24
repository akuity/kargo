//go:build e2e
//nolint:forcetypeassert
package git_helm_image

// This test implements the Git driven, Helm-based image-only example from
// https://github.com/akuity/kargo-examples
// (02-git-driven/02-helm-driven/01-image-only). Kargo watches a container
// image repository for new versions, renders the kargo-demo Helm chart with the
// updated image tag, and advances the result from stage to stage by committing
// the rendered manifests to a stage-specific branch, then pointing the Argo CD
// Application at the pushed commit.
//
// The prod stage opens a pull request and waits for it to be merged
// (git-open-pr / git-wait-for-pr); the test merges that PR with the configured
// PAT so the promotion can complete. AnalysisTemplate verification is stripped
// (see testdata/review/verification.yaml).

import (
	"testing"
	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestGitHelmImage(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
