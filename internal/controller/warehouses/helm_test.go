package warehouses

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
)

func TestDiscoverCharts(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		subs       []kargoapi.RepoSubscription
		assertions func(*testing.T, []kargoapi.ChartDiscoveryResult, error)
	}{
		{
			name:       "no chart subscription",
			reconciler: &reconciler{},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Empty(t, results)
			},
		},
		{
			name: "error obtaining credentials",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return nil, fmt.Errorf("something went wrong")
					},
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Chart: &kargoapi.ChartSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.Error(t, err)
				require.Empty(t, results)
			},
		},
		{
			name: "discovers chart versions",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverChartVersionsFn: func(
					context.Context,
					string,
					string,
					string,
					*helm.Credentials,
				) ([]string, error) {
					return []string{"1.1.0", "1.0.0"}, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Chart: &kargoapi.ChartSubscription{
					RepoURL: "https://example.com",
					Name:    "fake-chart",
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.ChartDiscoveryResult{
					{
						RepoURL:  "https://example.com",
						Name:     "fake-chart",
						Versions: []string{"1.1.0", "1.0.0"},
					},
				}, results)
			},
		},
		{
			name: "no chart versions discovered",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverChartVersionsFn: func(
					context.Context,
					string,
					string,
					string,
					*helm.Credentials,
				) ([]string, error) {
					return nil, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Chart: &kargoapi.ChartSubscription{
					RepoURL: "https://example.com",
					Name:    "fake-chart",
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.ChartDiscoveryResult{
					{
						RepoURL: "https://example.com",
						Name:    "fake-chart",
					},
				}, results)
			},
		},
		{
			name: "error discovering chart versions",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverChartVersionsFn: func(
					context.Context,
					string,
					string,
					string,
					*helm.Credentials,
				) ([]string, error) {
					return nil, fmt.Errorf("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Chart: &kargoapi.ChartSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.ErrorContains(t, err, "error discovering latest chart versions")
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, results)
			},
		},
		{
			name: "error discovering chart versions with chart name",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverChartVersionsFn: func(
					context.Context,
					string,
					string,
					string,
					*helm.Credentials,
				) ([]string, error) {
					return nil, fmt.Errorf("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Chart: &kargoapi.ChartSubscription{
					Name: "fake-chart",
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.ChartDiscoveryResult, err error) {
				require.ErrorContains(t, err, "error discovering latest chart versions for chart")
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, results)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := testCase.reconciler.discoverCharts(
				context.TODO(),
				"fake-namespace",
				testCase.subs,
			)
			testCase.assertions(t, results, err)
		})
	}
}
