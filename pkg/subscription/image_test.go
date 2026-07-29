package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/image"
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

func Test_imageSubscriber_DiscoverArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		subscriber *imageSubscriber
		sub        kargoapi.RepoSubscription
		assertions func(*testing.T, any, error)
	}{
		{
			name:       "no image subscription",
			subscriber: &imageSubscriber{},
			sub:        kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				require.Nil(t, res)
			},
		},
		{
			name: "error obtaining credentials",
			subscriber: &imageSubscriber{
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
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error obtaining credentials for image repo")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			// Credentials found for the repository are passed along to the Selector.
			name: "credentials are passed to selector",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						_ context.Context,
						namespace string,
						credType credentials.Type,
						repo string,
					) (*credentials.Credentials, error) {
						require.Equal(t, "fake-project", namespace)
						require.Equal(t, credentials.TypeImage, credType)
						require.Equal(t, "fake-url", repo)
						return &credentials.Credentials{
							Username: "fake-user",
							Password: "fake-password",
						}, nil
					},
				},
				newSelectorFn: func(
					_ context.Context,
					_ kargoapi.ImageSubscription,
					creds *image.Credentials,
				) (image.Selector, error) {
					require.NotNil(t, creds)
					require.Equal(t, "fake-user", creds.Username)
					require.Equal(t, "fake-password", creds.Password)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Absent credentials, the Selector receives none. It is expected to cope
			// with anonymous access to the repository.
			name: "no credentials found",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					_ context.Context,
					_ kargoapi.ImageSubscription,
					creds *image.Credentials,
				) (image.Selector, error) {
					require.Nil(t, creds)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Caching by tag being forbidden is silently enforced -- the Selector is
			// handed a subscription that has been stripped of the opt-in.
			name: "caching by tag is forbidden",
			subscriber: &imageSubscriber{
				credentialsDB:    &credentials.FakeDB{},
				cacheByTagPolicy: CacheByTagPolicyForbid,
				newSelectorFn: func(
					_ context.Context,
					sub kargoapi.ImageSubscription,
					_ *image.Credentials,
				) (image.Selector, error) {
					require.False(t, sub.CacheByTag)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{
				RepoURL:    "fake-url",
				CacheByTag: true,
			}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "caching by tag is allowed",
			subscriber: &imageSubscriber{
				credentialsDB:    &credentials.FakeDB{},
				cacheByTagPolicy: CacheByTagPolicyAllow,
				newSelectorFn: func(
					_ context.Context,
					sub kargoapi.ImageSubscription,
					_ *image.Credentials,
				) (image.Selector, error) {
					require.True(t, sub.CacheByTag)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{
				RepoURL:    "fake-url",
				CacheByTag: true,
			}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "caching by tag is required, but not opted into",
			subscriber: &imageSubscriber{
				credentialsDB:    &credentials.FakeDB{},
				cacheByTagPolicy: CacheByTagPolicyRequire,
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "caching image metadata by tag is required")
				require.Nil(t, res)
			},
		},
		{
			name: "caching by tag is required and opted into",
			subscriber: &imageSubscriber{
				credentialsDB:    &credentials.FakeDB{},
				cacheByTagPolicy: CacheByTagPolicyRequire,
				newSelectorFn: func(
					_ context.Context,
					sub kargoapi.ImageSubscription,
					_ *image.Credentials,
				) (image.Selector, error) {
					require.True(t, sub.CacheByTag)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{
				RepoURL:    "fake-url",
				CacheByTag: true,
			}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Caching by tag being forced is silently enforced -- the Selector is
			// handed a subscription that has been opted in on the user's behalf.
			name: "caching by tag is forced",
			subscriber: &imageSubscriber{
				credentialsDB:    &credentials.FakeDB{},
				cacheByTagPolicy: CacheByTagPolicyForce,
				newSelectorFn: func(
					_ context.Context,
					sub kargoapi.ImageSubscription,
					_ *image.Credentials,
				) (image.Selector, error) {
					require.True(t, sub.CacheByTag)
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, _ any, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error obtaining selector",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) (image.Selector, error) {
					return nil, errors.New("something went wrong")
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error obtaining selector for image")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			name: "error selecting images",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) (image.Selector, error) {
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return nil, errors.New("something went wrong")
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.ErrorContains(t, err, "error discovering newest applicable images")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, res)
			},
		},
		{
			name: "success -- named subscription",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) (image.Selector, error) {
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{
				Name: "fake-sub",
				Image: &kargoapi.ImageSubscription{
					RepoURL:  "fake-url",
					Platform: "linux/amd64",
				},
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.ImageDiscoveryResult{
						RepoURL:          "fake-url",
						Platform:         "linux/amd64",
						References:       []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}},
						SubscriptionName: "fake-sub",
					},
					res,
				)
			},
		},
		{
			name: "success -- unnamed subscription",
			subscriber: &imageSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) (image.Selector, error) {
					return &fakeImageSelector{
						selectFn: func(context.Context) ([]kargoapi.DiscoveredImageReference, error) {
							return []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}, nil
						},
					}, nil
				},
			},
			sub: kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"}},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.ImageDiscoveryResult)
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

// fakeImageSelector is a fake implementation of image.Selector for testing the
// imageSubscriber's discovery orchestration.
type fakeImageSelector struct {
	selectFn func(context.Context) ([]kargoapi.DiscoveredImageReference, error)
}

func (f *fakeImageSelector) MatchesTag(string) bool { return true }

func (f *fakeImageSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredImageReference, error) {
	return f.selectFn(ctx)
}
