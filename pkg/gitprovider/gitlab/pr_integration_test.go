//go:build integration && gitlab

package gitlab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
)

func TestCreateAndMergePullRequest(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_GITLAB_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_GITLAB_TOKEN")

	repoCfg := gptest.RepoConfig{
		RepoURL:           repoURL,
		Token:             token,
		GitUsername:       gptest.RequireEnv(t, "TEST_GITLAB_USERNAME"),
		MergeWaitDuration: 10 * time.Second,
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	gptest.RunPRTests(t, repoCfg, prov, []gptest.PRTestCase{
		{
			Name:            "unspecified merge method",
			ExpectedParents: 2,
		},
		{
			Name:            "merge",
			MergeMethod:     "merge",
			ExpectedParents: 2,
		},
		{
			Name:            "squash",
			MergeMethod:     "squash",
			ExpectedParents: 2, // GitLab squash still produces a merge commit
		},
		{
			Name:           "unsupported merge method",
			MergeMethod:    "rebase",
			ExpectMergeErr: true,
		},
	})
}
