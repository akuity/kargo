//go:build e2e
//nolint:forcetypeassert
package git_steps

// This suite exercises pull-request and tag git promotion steps with no Argo CD
// -- only git steps. Modeled on the prod stage of git_commit_only, it has one
// stage per step under test, each promoting freight directly from the Warehouse:
//
//   merge-pr    -- opens a PR and merges it with git-merge-pr (within the promotion)
//   wait-for-pr -- opens a PR and blocks on git-wait-for-pr until the test merges it
//   tag         -- creates and pushes an annotated tag with git-tag

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestGitSteps(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
