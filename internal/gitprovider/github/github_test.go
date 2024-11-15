package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGitHubURL(t *testing.T) {
	testCases := []struct {
		url           string
		expectedHost  string
		expectedOwner string
		expectedRepo  string
		errExpected   bool
	}{
		{
			url:         "not-a-url",
			errExpected: true,
		},
		{
			url:         "https://github.com/akuity",
			errExpected: true,
		},
		{
			url:           "https://github.com/akuity/kargo",
			expectedHost:  "github.com",
			expectedOwner: "akuity",
			expectedRepo:  "kargo",
		},
		{
			url:           "https://github.com/akuity/kargo.git",
			expectedHost:  "github.com",
			expectedOwner: "akuity",
			expectedRepo:  "kargo",
		},
		{
			// This isn't a real URL. It's just to validate that the function can
			// handle GitHub Enterprise URLs.
			url:           "https://github.akuity.io/akuity/kargo.git",
			expectedHost:  "github.akuity.io",
			expectedOwner: "akuity",
			expectedRepo:  "kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			host, owner, repo, err := parseRepoURL(testCase.url)
			if testCase.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedHost, host)
				require.Equal(t, testCase.expectedOwner, owner)
				require.Equal(t, testCase.expectedRepo, repo)
			}
		})
	}
}
