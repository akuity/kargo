//go:build e2e
//nolint:forcetypeassert
package git_kustomize_image

// This test implements the Git driven, kustomize image-only example from
// https://github.com/akuity/kargo-examples
// (02-git-driven/03-kustomize-driven/01-image-only/01-basic).
// Kargo watches a container image repository for new versions and advances them
// from stage to stage by rendering the kustomize base with the new image to a
// stage-specific branch, then pointing the Argo CD Application at the pushed
// commit.
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

func TestGitKustomizeImage(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
