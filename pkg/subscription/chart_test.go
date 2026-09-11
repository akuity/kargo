package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/helm"
	"github.com/akuity/kargo/pkg/helm/chart"
)

func Test_chartSubscriber_ApplySubscriptionDefaults(t *testing.T) {
	s := &chartSubscriber{}

	t.Run("preserves non-zero discoveryLimit", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{DiscoveryLimit: 13}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, int64(13), sub.Chart.DiscoveryLimit)
	})

	t.Run("no-op on nil chart", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Nil(t, sub.Chart)
	})
}

func Test_helmRepoURLRegex(t *testing.T) {
	cases := map[string]bool{
		"":                                 false,
		"ftp://charts.example":             false,
		"http://":                          false,
		"http://charts.example/path":       true,
		"https://":                         false,
		"https://charts example":           false,
		"https://charts.example":           true,
		"https://charts.example/":          true,
		"https://charts.example/path":      true,
		"https://charts.example:8080/path": true,
		"oci://":                           false,
		"oci://ghcr.io/org/chart":          true,
		"oci://ghcr.io/org/chart/":         true,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, helmRepoURLRegex.MatchString(input))
		})
	}
}

func Test_chartSubscriber_ValidateSubscription(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "RepoURL empty",
			sub: kargoapi.ChartSubscription{
				RepoURL: "",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "RepoURL invalid format",
			sub: kargoapi.ChartSubscription{
				RepoURL: "bogus-url",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "invalid semverConstraint",
			sub: kargoapi.ChartSubscription{
				RepoURL:          "https://charts.example.com",
				Name:             "mychart",
				SemverConstraint: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.semverConstraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "oci repoURL with name",
			sub: kargoapi.ChartSubscription{
				RepoURL: "oci://ghcr.io/example/chart",
				Name:    "should-not-be-here",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.name", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Contains(t, errs[0].Detail, "oci://")
			},
		},
		{
			name: "https repoURL without name",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://charts.example.com",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.name", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Contains(t, errs[0].Detail, "https://")
			},
		},
		{
			name: "http repoURL without name",
			sub: kargoapi.ChartSubscription{
				RepoURL: "http://charts.example.com",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.name", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Contains(t, errs[0].Detail, "http://")
			},
		},
		{
			name: "DiscoveryLimit lower bound is not validated",
			sub: kargoapi.ChartSubscription{
				RepoURL:        "oci://ghcr.io/example/chart",
				DiscoveryLimit: 0,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "DiscoveryLimit too large",
			sub: kargoapi.ChartSubscription{
				RepoURL:        "oci://ghcr.io/example/chart",
				DiscoveryLimit: 101,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "valid oci chart",
			sub: kargoapi.ChartSubscription{
				RepoURL:          "oci://ghcr.io/example/chart",
				SemverConstraint: "^1.0.0",
				DiscoveryLimit:   20,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "valid https chart",
			sub: kargoapi.ChartSubscription{
				RepoURL:          "https://charts.example.com",
				Name:             "mychart",
				SemverConstraint: "^1.0.0",
				DiscoveryLimit:   20,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	s := &chartSubscriber{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				s.ValidateSubscription(
					t.Context(),
					field.NewPath("chart"),
					kargoapi.RepoSubscription{Chart: &testCase.sub},
				),
			)
		})
	}
}

func Test_chartSubscriber_DiscoverArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		subscriber *chartSubscriber
		sub        kargoapi.RepoSubscription
		assertions func(t *testing.T, res any, err error)
	}{
		{
			name:       "no chart subscription",
			subscriber: &chartSubscriber{},
			sub:        kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				require.Nil(t, res)
			},
		},
		{
			name: "error obtaining credentials",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return nil, errors.New("something went wrong")
					},
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error obtaining credentials for chart repository")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			// Credentials found for the repository are passed along to the Selector.
			name: "credentials are passed to selector",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						_ context.Context,
						namespace string,
						credType credentials.Type,
						repo string,
					) (*credentials.Credentials, error) {
						require.Equal(t, "fake-project", namespace)
						require.Equal(t, credentials.TypeHelm, credType)
						require.Equal(t, "fake-url", repo)
						return &credentials.Credentials{
							Username: "fake-user",
							Password: "fake-password",
						}, nil
					},
				},
				newSelectorFn: func(
					_ context.Context,
					_ kargoapi.ChartSubscription,
					_ int,
					creds *helm.Credentials,
				) (chart.Selector, error) {
					require.NotNil(t, creds)
					require.Equal(t, "fake-user", creds.Username)
					require.Equal(t, "fake-password", creds.Password)
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Absent credentials, the Selector receives none. It is expected to cope
			// with anonymous access to the repository.
			name: "no credentials found",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					_ context.Context,
					_ kargoapi.ChartSubscription,
					_ int,
					creds *helm.Credentials,
				) (chart.Selector, error) {
					require.Nil(t, creds)
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error obtaining selector",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return nil, errors.New("something went wrong")
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error obtaining selector for chart versions")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			name: "error selecting versions",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return nil, errors.New("something went wrong")
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error discovering chart versions")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			// The Selector is entrusted with the whole subscription, discovery limit
			// included, but the limit is enforced here regardless.
			name: "discovered versions are trimmed to the discovery limit",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return []string{"3.0.0", "2.0.0", "1.0.0"}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{
				RepoURL:        "fake-url",
				DiscoveryLimit: 2,
			}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.ChartDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, []string{"3.0.0", "2.0.0"}, result.Versions)
			},
		},
		{
			// The Selector is entrusted with the whole subscription, discovery limit
			// included, but the limit is enforced here regardless.
			name: "discovered versions are trimmed to the repo sub discovery limit",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return []string{"3.0.0", "2.0.0", "1.0.0"}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{
				DiscoveryLimit: 2,
				Chart: &kargoapi.ChartSubscription{
					RepoURL: "fake-url",
				}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.ChartDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, []string{"3.0.0", "2.0.0"}, result.Versions)
			},
		},
		{
			name: "success -- named subscription",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return []string{"1.1.0", "1.0.0"}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{
				Name:           "fake-sub",
				DiscoveryLimit: 20,
				Chart: &kargoapi.ChartSubscription{
					RepoURL:          "fake-url",
					Name:             "fake-chart",
					SemverConstraint: "^1.0.0",
				},
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.ChartDiscoveryResult{
						RepoURL:          "fake-url",
						Name:             "fake-chart",
						SemverConstraint: "^1.0.0",
						Versions:         []string{"1.1.0", "1.0.0"},
						SubscriptionName: "fake-sub",
					},
					res,
				)
			},
		},
		{
			name: "success -- unnamed subscription",
			subscriber: &chartSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ChartSubscription,
					int,
					*helm.Credentials,
				) (chart.Selector, error) {
					return &fakeChartSelector{
						selectFn: func(context.Context) ([]string, error) {
							return []string{"1.0.0"}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{
				DiscoveryLimit: 20,
				Chart: &kargoapi.ChartSubscription{
					RepoURL: "fake-url",
				}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.ChartDiscoveryResult)
				require.True(t, ok)
				require.Empty(t, result.SubscriptionName)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.subscriber.DiscoverArtifacts(
				t.Context(),
				"fake-project",
				testCase.sub,
				nil,
			)
			testCase.assertions(t, res, err)
		})
	}
}

// fakeChartSelector is a fake implementation of chart.Selector for testing the
// chartSubscriber's discovery orchestration.
type fakeChartSelector struct {
	selectFn func(context.Context) ([]string, error)
}

func (f *fakeChartSelector) MatchesVersion(string) bool { return true }

func (f *fakeChartSelector) Select(ctx context.Context) ([]string, error) {
	return f.selectFn(ctx)
}
