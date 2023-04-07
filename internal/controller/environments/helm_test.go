package environments

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
)

func TestGetLatestCharts(t *testing.T) {
	testCases := []struct {
		name                    string
		credentialsDB           credentials.Database
		getLatestChartVersionFn func(
			context.Context,
			string,
			string,
			string,
			*helm.Credentials,
		) (string, error)
		assertions func([]api.Chart, error)
	}{
		{
			name: "error getting registry credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false,
						errors.New("something went wrong")
				},
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for chart registry",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest chart version",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error searching for latest version of chart",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "no chart found",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", nil
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no suitable version of chart")
			},
		},

		{
			name: "success",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "1.0.0", nil
			},
			assertions: func(charts []api.Chart, err error) {
				require.NoError(t, err)
				require.Len(t, charts, 1)
				require.Equal(
					t,
					api.Chart{
						RegistryURL: "fake-url",
						Name:        "fake-chart",
						Version:     "1.0.0",
					},
					charts[0],
				)
			},
		},
	}
	testSubs := []api.ChartSubscription{
		{
			RegistryURL: "fake-url",
			Name:        "fake-chart",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := reconciler{
				credentialsDB:           testCase.credentialsDB,
				getLatestChartVersionFn: testCase.getLatestChartVersionFn,
			}
			testCase.assertions(
				reconciler.getLatestCharts(
					context.Background(),
					"fake-namespace",
					testSubs,
				),
			)
		})
	}
}
