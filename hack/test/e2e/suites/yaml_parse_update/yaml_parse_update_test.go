//go:build e2e
//nolint:forcetypeassert
package yaml_parse_update

// This test is adapted from the yaml-parse / yaml-update example at
// https://github.com/akuity/kargo-examples (03-features/03-yaml-parse-update).
//
// Differences from the source example:
//   - It uses the shared kargo-demo-gitops repository (like the other suites)
//     instead of the kargo-examples repository, parsing/updating image.name in
//     charts/kargo-demo/values.yaml on the new-helm branch.
//   - The git-commit and git-push steps are removed, so the working copy is
//     never written back to the repository.
//   - A second yaml-parse step re-reads the updated field, and the test asserts
//     that yaml-update actually changed it.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestYAMLParseUpdate(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
