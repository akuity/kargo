package azure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRepoURL(t *testing.T) {
	testCases := []struct {
		name         string
		url          string
		expectedOrg  string
		expectedProj string
		expectedRepo string
		errExpected  bool
	}{
		{
			name:        "invalid URL",
			url:         "not-a-url",
			errExpected: true,
		},
		{
			name:        "unsupported host",
			url:         "https://github.com/org/repo",
			errExpected: true,
		},
		{
			name:        "modern URL with missing parts",
			url:         "https://dev.azure.com/org",
			errExpected: true,
		},
		{
			name:        "legacy URL with missing parts",
			url:         "https://org.visualstudio.com",
			errExpected: true,
		},
		{
			name:         "modern URL format",
			url:          "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "modern URL format with .git suffix",
			url:          "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "legacy URL format",
			url:          "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "legacy URL format with .git suffix",
			url:          "https://myorg.visualstudio.com/myproject/_git/myrepo.git",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "modern URL format with dot in repo name",
			url:          "https://dev.azure.com/myorg/myproject/_git/my.repo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "my.repo",
			errExpected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			org, proj, repo, err := parseRepoURL(tc.url)
			if tc.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOrg, org)
				require.Equal(t, tc.expectedProj, proj)
				require.Equal(t, tc.expectedRepo, repo)
			}
		})
	}
}

func TestGetCommitURL(t *testing.T) {

	testCases := []struct {
		url         string
		sha         string
		expectedURL string
	}{
		{
			url:         "ssh://git@ssh.dev.azure.com:v3/akuity/_git/kargo",
			sha:         "sha",
			expectedURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
		{
			url:         "git@ssh.dev.azure.com:v3/akuity/_git/kargo",
			sha:         "sha",
			expectedURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
		{
			url:         "http://dev.azure.com/akuity/_git/kargo",
			sha:         "sha",
			expectedURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
	}

	for _, testCase := range testCases {
		// call the code we are testing
		g := provider{}
		commitURL, err := g.GetCommitURL(testCase.url, testCase.sha)
		require.NoError(t, err)
		require.Equal(t, testCase.expectedURL, commitURL)
	}
}
