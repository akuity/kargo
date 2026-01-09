package subscription

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_imageSubscriber_ApplySubscriptionDefaults(t *testing.T) {
	s := &imageSubscriber{}

	t.Run("defaults empty fields", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.ImageSelectionStrategySemVer, sub.Image.ImageSelectionStrategy)
		require.NotNil(t, sub.Image.StrictSemvers)
		require.True(t, *sub.Image.StrictSemvers)
		require.Equal(t, int64(20), sub.Image.DiscoveryLimit)
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		strict := false
		sub := &kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{
			ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
			StrictSemvers:          &strict,
			DiscoveryLimit:         9,
		}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.ImageSelectionStrategyNewestBuild, sub.Image.ImageSelectionStrategy)
		require.NotNil(t, sub.Image.StrictSemvers)
		require.False(t, *sub.Image.StrictSemvers)
		require.Equal(t, int64(9), sub.Image.DiscoveryLimit)
	})

	t.Run("no-op on nil image", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Nil(t, sub.Image)
	})
}

func Test_imageRepoURLRegex(t *testing.T) {
	cases := map[string]bool{
		"":                              false,
		"repo":                          true,
		"library/ubuntu":                true,
		"/akuity/kargo":                 false,
		"docker.io/library/ubuntu":      true,
		"ghcr.io/akuity/kargo":          true,
		"ghcr.io/akuity/kargo/sub":      true,
		"ghcr.io/akuity/kargo-sub":      true,
		"ghcr.io/akuity/kargo.sub":      true,
		"ghcr.io:443/akuity/kargo":      true,
		"ghcr.io/akuity/kargo/":         false,
		"ghcr.io//akuity/kargo":         false,
		"ghcr.io/akuity//kargo":         false,
		"ghcr.io/akuity/kargo@sha256":   false,
		"ghcr.io/akuity/kargo:tag":      false,
		"ghcr.io/akuity/kargo:tag/name": false,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, imageRepoURLRegex.MatchString(input))
		})
	}
}

func Test_imageSubscriber_ValidateSubscription(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "RepoURL empty",
			sub: kargoapi.ImageSubscription{
				RepoURL: "",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "RepoURL invalid format",
			sub: kargoapi.ImageSubscription{
				RepoURL: "bogus invalid",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "ImageSelectionStrategy invalid",
			sub: kargoapi.ImageSubscription{
				RepoURL:                "ghcr.io/akuity/kargo",
				ImageSelectionStrategy: "InvalidStrategy",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.imageSelectionStrategy", errs[0].Field)
				require.Equal(t, field.ErrorTypeNotSupported, errs[0].Type)
			},
		},
		{
			name: "Digest strategy missing constraint",
			sub: kargoapi.ImageSubscription{
				RepoURL:                "ghcr.io/akuity/kargo",
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
				Constraint:             "",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.constraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "invalid constraint for SemVer",
			sub: kargoapi.ImageSubscription{
				RepoURL:                "ghcr.io/akuity/kargo",
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.constraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "Platform invalid format",
			sub: kargoapi.ImageSubscription{
				RepoURL:  "ghcr.io/akuity/kargo",
				Platform: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.platform", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "caching by tag required but not set",
			sub: kargoapi.ImageSubscription{
				RepoURL: "ghcr.io/akuity/kargo",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.cacheByTag", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too small",
			sub: kargoapi.ImageSubscription{
				RepoURL:        "ghcr.io/akuity/kargo",
				CacheByTag:     true,
				DiscoveryLimit: 0,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too large",
			sub: kargoapi.ImageSubscription{
				RepoURL:        "ghcr.io/akuity/kargo",
				CacheByTag:     true,
				DiscoveryLimit: 101,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "image.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "valid",
			sub: kargoapi.ImageSubscription{
				RepoURL:                "ghcr.io/akuity/kargo",
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
				Constraint:             "^1.0.0",
				Platform:               "linux/amd64",
				CacheByTag:             true,
				DiscoveryLimit:         20,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	s := &imageSubscriber{
		cacheByTagPolicy: CacheByTagPolicyRequire,
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				s.ValidateSubscription(
					t.Context(),
					field.NewPath("image"),
					kargoapi.RepoSubscription{Image: &testCase.sub},
				),
			)
		})
	}
}
