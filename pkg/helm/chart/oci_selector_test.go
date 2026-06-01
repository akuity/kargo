package chart

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewOCISelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building base selector",
			sub: kargoapi.ChartSubscription{
				SemverConstraint: "invalid", // This will force an error
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			name: "error parsing repo url",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://charts.example.com", // Not OCI
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing repository URL")
			},
		},
		{
			name: "success",
			sub: kargoapi.ChartSubscription{
				RepoURL: "oci://charts.example.com/repo",
				Name:    "my-chart",
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				o, ok := s.(*ociSelector)
				require.True(t, ok)
				require.NotNil(t, o.baseSelector)
				require.NotNil(t, o.repo)
			},
		},
		{
			name: "insecureSkipTLSVerify propagated to authorizer",
			sub: kargoapi.ChartSubscription{
				RepoURL:               "oci://charts.example.com/repo",
				InsecureSkipTLSVerify: true,
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				o, ok := s.(*ociSelector)
				require.True(t, ok)
				require.True(t, o.insecureSkipTLSVerify)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newOCISelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_ociSelector_Select(t *testing.T) {
	// Instead of mocking out an OCI registry, it's more expedient to use Kargo's
	// own chart repo on ghcr.io to test this.
	s, err := newOCISelector(
		kargoapi.ChartSubscription{
			RepoURL: "oci://ghcr.io/akuity/kargo-charts/kargo",
		},
		nil,
	)
	require.NoError(t, err)
	versions, err := s.Select(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, versions)
}

func Test_parseOCITagSemver(t *testing.T) {
	testCases := []struct {
		name     string
		tag      string
		expected string
		assert   func(*testing.T, error)
	}{
		{
			name:     "strict semver",
			tag:      "1.2.3",
			expected: "1.2.3",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "helm build metadata tag",
			tag:      "1.2.3_build.1",
			expected: "1.2.3+build.1",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "prerelease",
			tag:      "1.2.3-rc.1",
			expected: "1.2.3-rc.1",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "v-prefixed semver",
			tag:      "v1.2.3",
			expected: "1.2.3",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "v-prefixed helm build metadata tag",
			tag:      "v1.2.3_build.1",
			expected: "1.2.3+build.1",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "partial semver",
			tag:  "1.2",
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "non-semver tag",
			tag:  "latest",
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sv, err := parseOCITagSemver(testCase.tag)
			testCase.assert(t, err)
			if testCase.expected != "" {
				require.Equal(t, testCase.expected, sv.Original())
			}
		})
	}
}
