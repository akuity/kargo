package subscription

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_gitSubscriber_ApplySubscriptionDefaults(t *testing.T) {
	s := &gitSubscriber{}

	t.Run("defaults empty fields", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.CommitSelectionStrategyNewestFromBranch, sub.Git.CommitSelectionStrategy)
		require.NotNil(t, sub.Git.StrictSemvers)
		require.True(t, *sub.Git.StrictSemvers)
		require.Equal(t, int64(20), sub.Git.DiscoveryLimit)
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		strict := false
		sub := &kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{
			CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
			StrictSemvers:           &strict,
			DiscoveryLimit:          7,
		}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.CommitSelectionStrategySemVer, sub.Git.CommitSelectionStrategy)
		require.NotNil(t, sub.Git.StrictSemvers)
		require.False(t, *sub.Git.StrictSemvers)
		require.Equal(t, int64(7), sub.Git.DiscoveryLimit)
	})

	t.Run("no-op on nil git", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{}
		// Should not panic
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Nil(t, sub.Git)
	})
}

func Test_gitURLRegex(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		":":            false,
		"/etc/passwd":  false,
		"//etc/passwd": false,
		"https:":       false,

		"https://not a url":                      false,
		"http://github.com/example/repo?foo=bar": true,
		"ssh://not a url":                        false,
		"ssh://github.com/example/repo?foo=bar":  true,
		"not even remotely a url":                false,
		// URLs of the form http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
		"https://github.com":          false,
		"https://github.com/":         true,
		"https://foo:bar@github.com":  false,
		"https://foo:bar@github.com/": true,
		// Variable features
		"https://github.com/example/repo":                  true,
		"https://github.com/example/repo/":                 true,
		"https://github.com/example/repo.git":              true,
		"https://github.com/example/repo.git/":             true,
		"https://localhost:8443/example/repo":              true,
		"https://localhost:8443/example/repo/":             true,
		"https://localhost:8443/example/repo.git":          true,
		"https://localhost:8443/example/repo.git/":         true,
		"https://foo:bar@github.com/example/repo":          true,
		"https://foo:bar@github.com/example/repo/":         true,
		"https://foo:bar@github.com/example/repo.git":      true,
		"https://foo:bar@github.com/example/repo.git/":     true,
		"https://foo:bar@localhost:8443/example/repo":      true,
		"https://foo:bar@localhost:8443/example/repo/":     true,
		"https://foo:bar@localhost:8443/example/repo.git":  true,
		"https://foo:bar@localhost:8443/example/repo.git/": true,
		// ssh forms
		"ssh://git.example.com":      false,
		"ssh://git.example.com/":     true,
		"ssh://git@git.example.com":  false,
		"ssh://git@git.example.com/": true,
		// Variable features
		"ssh://github.com/example/repo":              true,
		"ssh://github.com/example/repo/":             true,
		"ssh://github.com/example/repo.git":          true,
		"ssh://github.com/example/repo.git/":         true,
		"ssh://localhost:2222/example/repo":          true,
		"ssh://localhost:2222/example/repo/":         true,
		"ssh://localhost:2222/example/repo.git":      true,
		"ssh://localhost:2222/example/repo.git/":     true,
		"ssh://git@github.com/example/repo":          true,
		"ssh://git@github.com/example/repo/":         true,
		"ssh://git@github.com/example/repo.git":      true,
		"ssh://git@github.com/example/repo.git/":     true,
		"ssh://git@localhost:2222/example/repo":      true,
		"ssh://git@localhost:2222/example/repo/":     true,
		"ssh://git@localhost:2222/example/repo.git":  true,
		"ssh://git@localhost:2222/example/repo.git/": true,
		// SCP-style
		"git.example.com":     false,
		"git@git.example.com": false,
		// Variable features
		"github.com:example/repo":          false,
		"github.com:example/repo/":         false,
		"github.com:example/repo.git":      false,
		"github.com:example/repo.git/":     false,
		"git@github.com:example/repo":      true,
		"git@github.com:example/repo/":     true,
		"git@github.com:example/repo.git":  true,
		"git@github.com:example/repo.git/": true,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, gitURLRegex.MatchString(input))
		})
	}
}

func Test_branchRegex(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		"foo/bar":      true,
		"foo.bar":      true,
		"release-0.58": true,
		"/foo":         false,
		"foo/":         false,
		"foo//bar":     true,
		".foo":         false,
		"foo.":         false,
		"foo..bar":     true,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, branchRegex.MatchString(input))
		})
	}
}

func Test_gitSubscriber_ValidateSubscription(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "RepoURL empty",
			sub: kargoapi.GitSubscription{
				RepoURL: "",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "RepoURL invalid format",
			sub: kargoapi.GitSubscription{
				RepoURL: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "Branch too long",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
				Branch:  "a" + strings.Repeat("b", 255),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.branch", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "Branch invalid pattern",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
				Branch:  "-invalid",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.branch", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "CommitSelectionStrategy invalid",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				CommitSelectionStrategy: "InvalidStrategy",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.commitSelectionStrategy", errs[0].Field)
				require.Equal(t, field.ErrorTypeNotSupported, errs[0].Type)
			},
		},
		{
			name: "invalid semverConstraint",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				SemverConstraint: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.semverConstraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too small",
			sub: kargoapi.GitSubscription{
				RepoURL:        "https://github.com/akuity/kargo.git",
				DiscoveryLimit: 0,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too large",
			sub: kargoapi.GitSubscription{
				RepoURL:        "https://github.com/akuity/kargo.git",
				DiscoveryLimit: 101,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "valid",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				Branch:                  "main",
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				SemverConstraint:        "^1.0.0",
				DiscoveryLimit:          20,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	s := &gitSubscriber{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				s.ValidateSubscription(
					t.Context(),
					field.NewPath("git"),
					kargoapi.RepoSubscription{Git: &testCase.sub},
				),
			)
		})
	}
}
