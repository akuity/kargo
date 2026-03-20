package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRepoURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		url    string
		assert func(*testing.T, string, string, string, string, error)
	}{
		{
			name: "standard GitHub URL",
			url:  "https://github.com/akuity/kargo",
			assert: func(t *testing.T, scheme, host, owner, repo string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https", scheme)
				require.Equal(t, "github.com", host)
				require.Equal(t, "akuity", owner)
				require.Equal(t, "kargo", repo)
			},
		},
		{
			name: ".git suffix",
			url:  "https://github.com/akuity/kargo.git",
			assert: func(t *testing.T, scheme, host, owner, repo string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https", scheme)
				require.Equal(t, "github.com", host)
				require.Equal(t, "akuity", owner)
				require.Equal(t, "kargo", repo)
			},
		},
		{
			name: "GitHub Enterprise URL",
			url:  "https://github.example.com/myorg/myrepo",
			assert: func(t *testing.T, scheme, host, owner, repo string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https", scheme)
				require.Equal(t, "github.example.com", host)
				require.Equal(t, "myorg", owner)
				require.Equal(t, "myrepo", repo)
			},
		},
		{
			name: "SSH URL with git@ prefix",
			url:  "git@github.com:akuity/kargo.git",
			assert: func(t *testing.T, scheme, host, owner, repo string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https", scheme)
				require.Equal(t, "github.com", host)
				require.Equal(t, "akuity", owner)
				require.Equal(t, "kargo", repo)
			},
		},
		{
			name: "invalid URL with wrong path segments",
			url:  "https://github.com/akuity",
			assert: func(t *testing.T, _, _, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(),
					"could not extract repository owner and name",
				)
			},
		},
		{
			name: "empty URL",
			url:  "",
			assert: func(t *testing.T, _, _, _, _ string, err error) {
				require.Error(t, err)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scheme, host, owner, repo, err := ParseRepoURL(tc.url)
			tc.assert(t, scheme, host, owner, repo, err)
		})
	}
}

func TestBuildCommitURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		repoURL  string
		sha      string
		expected string
	}{
		{
			name:     "standard URL",
			repoURL:  "https://github.com/akuity/kargo",
			sha:      "abc123",
			expected: "https://github.com/akuity/kargo/commit/abc123",
		},
		{
			name:     ".git suffix",
			repoURL:  "https://github.com/akuity/kargo.git",
			sha:      "abc123",
			expected: "https://github.com/akuity/kargo/commit/abc123",
		},
		{
			name:     "Enterprise URL",
			repoURL:  "https://github.example.com/myorg/myrepo",
			sha:      "def456",
			expected: "https://github.example.com/myorg/myrepo/commit/def456",
		},
		{
			name:     "invalid URL returns empty string",
			repoURL:  "https://github.com/akuity",
			sha:      "abc123",
			expected: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := BuildCommitURL(tc.repoURL, tc.sha)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		url    string
		token  string
		assert func(*testing.T, string, string, error)
	}{
		{
			name:  "standard URL returns correct owner and repo",
			url:   "https://github.com/akuity/kargo",
			token: "fake-token",
			assert: func(t *testing.T, owner, repo string, err error) {
				require.NoError(t, err)
				require.Equal(t, "akuity", owner)
				require.Equal(t, "kargo", repo)
			},
		},
		{
			name:  "invalid URL returns error",
			url:   "https://github.com/akuity",
			token: "",
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(),
					"could not extract repository owner and name",
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, owner, repo, err := NewClient(tc.url, tc.token, false)
			if err == nil {
				require.NotNil(t, client)
			}
			tc.assert(t, owner, repo, err)
		})
	}
}
