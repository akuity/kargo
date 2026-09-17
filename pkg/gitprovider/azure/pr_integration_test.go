//go:build integration && azure

package azure

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
)

func TestCreateAndMergePullRequest(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_AZURE_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_AZURE_TOKEN")
	gitUsername := gptest.RequireEnv(t, "TEST_AZURE_USERNAME")

	repoCfg := gptest.RepoConfig{
		RepoURL:     repoURL,
		Token:       token,
		GitUsername: gitUsername,
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	gptest.RunPRTests(t, repoCfg, prov, []gptest.PRTestCase{
		{
			Name:            "unspecified merge method",
			ExpectedParents: 2, // Default is noFastForward (merge commit)
		},
		{
			Name:            "noFastForward",
			MergeMethod:     "noFastForward",
			ExpectedParents: 2,
		},
		{
			Name:            "rebase",
			MergeMethod:     "rebase",
			ExpectedParents: 1,
		},
		{
			Name:            "rebaseMerge",
			MergeMethod:     "rebaseMerge",
			ExpectedParents: 2,
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
