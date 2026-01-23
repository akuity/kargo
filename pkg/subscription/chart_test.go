package subscription

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_chartSubscriber_ApplySubscriptionDefaults(t *testing.T) {
	s := &chartSubscriber{}

	t.Run("defaults discoveryLimit", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, int64(20), sub.Chart.DiscoveryLimit)
	})

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
			name: "DiscoveryLimit too small",
			sub: kargoapi.ChartSubscription{
				RepoURL:        "oci://ghcr.io/example/chart",
				DiscoveryLimit: 0,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "chart.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
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
