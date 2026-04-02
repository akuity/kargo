//go:build integration && bitbucket

package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
)

func TestCreateAndMergePullRequest(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_BITBUCKET_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_BITBUCKET_TOKEN")

	repoCfg := gptest.RepoConfig{
		RepoURL:     repoURL,
		Token:       token,
		GitUsername: gptest.RequireEnv(t, "TEST_BITBUCKET_USERNAME"),
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	gptest.RunPRTests(t, repoCfg, prov, []gptest.PRTestCase{
		{
			Name:            "unspecified merge method",
			ExpectedParents: 2, // Repo default is merge commit
		},
		{
			Name:           "explicit merge method",
			MergeMethod:    "squash",
			ExpectMergeErr: true, // Library limitation
		},
	})
}
