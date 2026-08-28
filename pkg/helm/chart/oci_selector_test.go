package chart

import (
	"context"
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
	versions, err := s.Select(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, versions)
}
