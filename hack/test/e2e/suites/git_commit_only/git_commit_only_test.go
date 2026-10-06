//go:build e2e
//nolint:forcetypeassert
package git_commit_only

// This test implements the Git driven, commit-only example from
// https://github.com/akuity/kargo-examples (02-git-driven/01-commit-only).
// Kargo watches the kustomize branch for new commits and advances them from
// stage to stage by copying select contents to the head of a stage-specific
// branch, then pointing the Argo CD Application at the pushed commit.
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

func TestGitCommitOnly(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
