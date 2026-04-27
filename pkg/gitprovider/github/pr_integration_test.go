//go:build integration && github

package github

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
)

func TestCreateAndMergePullRequest(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_GITHUB_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_GITHUB_TOKEN")

	repoCfg := gptest.RepoConfig{
		RepoURL:     repoURL,
		Token:       token,
		GitUsername: gptest.RequireEnv(t, "TEST_GITHUB_USERNAME"),
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	gptest.RunPRTests(t, repoCfg, prov, []gptest.PRTestCase{
		{
			Name:            "unspecified merge method",
			ExpectedParents: 2, // Default is merge commit
		},
		{
			Name:            "merge",
			MergeMethod:     "merge",
			ExpectedParents: 2,
		},
		{
			Name:            "rebase",
			MergeMethod:     "rebase",
			ExpectedParents: 1,
		},
		{
			Name:            "squash",
			MergeMethod:     "squash",
			ExpectedParents: 1,
		},
		{
			Name:           "invalid merge method",
			MergeMethod:    "bogus",
			ExpectMergeErr: true,
		},
	})
}
