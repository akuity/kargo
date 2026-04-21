package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRepoURL(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		expectedScheme string
		expectedHost   string
		expectedOwner  string
		expectedRepo   string
		errExpected    bool
	}{
		{
			name:        "invalid URL",
			url:         "not-a-url",
			errExpected: true,
		},
		{
			name:        "missing repo name",
			url:         "https://github.com/akuity",
			errExpected: true,
		},
		{
			name:           "standard HTTPS URL",
			url:            "https://github.com/akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			name:           "HTTPS URL with .git suffix",
			url:            "https://github.com/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			name:           "GitHub Enterprise URL",
			url:            "https://github.akuity.io/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "github.akuity.io",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			name:           "HTTP URL with port",
			url:            "http://git@example.com:8080/akuity/kargo",
			expectedScheme: "http",
			expectedHost:   "example.com:8080",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			name:           "SSH URL",
			url:            "git@github.com:akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme, host, owner, repo, err := ParseRepoURL(testCase.url)
			if testCase.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedScheme, scheme)
				require.Equal(t, testCase.expectedHost, host)
				require.Equal(t, testCase.expectedOwner, owner)
				require.Equal(t, testCase.expectedRepo, repo)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name        string
		repoURL     string
		opts        *ClientOptions
		errExpected bool
	}{
		{
			name:        "invalid URL",
			repoURL:     "not-a-url",
			errExpected: true,
		},
		{
			name:    "github.com with nil opts",
			repoURL: "https://github.com/akuity/kargo",
		},
		{
			name:    "github.com with token",
			repoURL: "https://github.com/akuity/kargo",
			opts:    &ClientOptions{Token: "test-token"},
		},
		{
			name:    "GitHub Enterprise",
			repoURL: "https://github.akuity.io/akuity/kargo",
			opts:    &ClientOptions{Token: "test-token"},
		},
		{
			name:    "insecure TLS skip",
			repoURL: "https://github.com/akuity/kargo",
			opts: &ClientOptions{
				InsecureSkipTLSVerify: true,
				Token:                 "test-token",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := NewClient(testCase.repoURL, testCase.opts)
			if testCase.errExpected {
				require.Error(t, err)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}
