package commit

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestNewBaseSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		creds      *git.RepoCredentials
		assertions func(*testing.T, *baseSelector, error)
	}{
		{
			name: "error parsing filter expression",
			sub:  kargoapi.GitSubscription{ExpressionFilter: "(1 + 2"},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error compiling filter expression")
			},
		},
		{
			name: "error parsing include path selectors",
			sub: kargoapi.GitSubscription{
				IncludePaths: []string{"regex:["}, // Bad regex
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error parsing include path selectors")
			},
		},
		{
			name: "error parsing exclude path selectors",
			sub: kargoapi.GitSubscription{
				ExcludePaths: []string{"regex:["}, // Bad regex
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error parsing exclude path selectors")
			},
		},
		{
			name: "success",
			sub: kargoapi.GitSubscription{
				RepoURL:               "https://github.com/example/repo.git",
				InsecureSkipTLSVerify: true,
				ExpressionFilter:      "false",
				IncludePaths:          []string{"apps/"},
				ExcludePaths:          []string{"hack/"},
				DiscoveryLimit:        5,
			},
			creds: &git.RepoCredentials{
				Username: "foo",
				Password: "bar",
			},
			assertions: func(t *testing.T, s *baseSelector, err error) {
				require.NoError(t, err)
				require.Equal(t, "https://github.com/example/repo.git", s.repoURL)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "foo",
						Password: "bar",
					},
					s.creds,
				)
				require.True(t, s.insecureSkipTLSVerify)
				require.NotNil(t, s.filterExpression)
				require.NotNil(t, s.includePaths)
				require.NotNil(t, s.excludePaths)
				require.Equal(t, 5, s.discoveryLimit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newBaseSelector(testCase.sub, testCase.creds)
			testCase.assertions(t, s, err)
		})
	}
}
