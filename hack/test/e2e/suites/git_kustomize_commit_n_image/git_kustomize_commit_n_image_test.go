//go:build e2e
//nolint:forcetypeassert
package git_kustomize_commit_n_image

// This test implements the Git driven, kustomize commit-n-image example from
// https://github.com/akuity/kargo-examples
// (02-git-driven/03-kustomize-driven/02-commit-n-image). Kargo watches the
// kustomize branch for new commits and a container image for new versions,
// advancing them from stage to stage by rendering manifests with the updated
// image and pushing the result to the head of a stage-specific branch, then
// pointing the Argo CD Application at the pushed commit.
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

func TestGitKustomizeCommitNImage(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
