package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_isMaintainer(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         config
		authorAssoc string
		login       string
		orgsClient  OrganizationsClient
		assert      func(*testing.T, bool, error)
	}{
		{
			name: "fast path: MEMBER matches",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "MEMBER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path: OWNER matches",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "OWNER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path: case insensitive",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "member",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path miss + no orgsClient: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient:  nil,
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "fast path miss + empty login: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					t.Fatal("IsMember should not be called without a login")
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "fast path miss + MEMBER not configured: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					t.Fatal("IsMember should not be called when MEMBER is not configured")
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "slow path: concealed member resolves as maintainer",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(_ context.Context, org, user string) (bool, *github.Response, error) {
					require.Equal(t, "akuity", org)
					require.Equal(t, "alice", user)
					return true, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "slow path: non-member is not a maintainer",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "bob",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "slow path: IsMember error propagates",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					return false, nil, errors.New("boom")
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.ErrorContains(t, err, "boom")
				require.False(t, ok)
			},
		},
		{
			name: "no association configured: nobody is a maintainer",
			cfg: config{
				MaintainerAssociations: nil,
			},
			authorAssoc: "MEMBER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ok, err := isMaintainer(
				context.Background(),
				testCase.cfg,
				"akuity",
				testCase.authorAssoc,
				testCase.login,
				testCase.orgsClient,
			)
			testCase.assert(t, ok, err)
		})
	}
}

func Test_needsLabel(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		existingLabels map[string]struct{}
		expected       bool
	}{
		{
			name:           "label present",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kind/bug": {}},
			expected:       false,
		},
		{
			name:           "label missing",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"priority/high": {}},
			expected:       true,
		},
		{
			name:           "no labels at all",
			prefix:         "kind",
			existingLabels: map[string]struct{}{},
			expected:       true,
		},
		{
			name:           "prefix without slash does not match",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kinder": {}},
			expected:       true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := needsLabel(testCase.prefix, testCase.existingLabels)
			require.Equal(t, testCase.expected, result)
		})
	}
}
