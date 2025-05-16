package warehouses

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/image"
)

func TestDiscoverImages(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		subs       []kargoapi.RepoSubscription
		assertions func(*testing.T, []kargoapi.ImageDiscoveryResult, error)
	}{
		{
			name:       "no image subscription",
			reconciler: &reconciler{},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ImageDiscoveryResult, err error) {
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
				{Image: &kargoapi.ImageSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ImageDiscoveryResult, err error) {
				require.Error(t, err)
				require.Empty(t, results)
			},
		},
		{
			name: "discovers image references",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverImageRefsFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) ([]image.Image, error) {
					return []image.Image{
						{Tag: "xyz"},
						{Tag: "abc"},
					}, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Image: &kargoapi.ImageSubscription{
					RepoURL: "fake-repo",
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.ImageDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "fake-repo",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: "xyz"},
							{Tag: "abc"},
						},
					},
				}, results)
			},
		},
		{
			name: "error discovering image references",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverImageRefsFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) ([]image.Image, error) {
					return nil, fmt.Errorf("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Image: &kargoapi.ImageSubscription{}},
			},
			assertions: func(t *testing.T, results []kargoapi.ImageDiscoveryResult, err error) {
				require.Error(t, err)
				require.Empty(t, results)
			},
		},
		{
			name: "no suitable images discovered",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				discoverImageRefsFn: func(
					context.Context,
					kargoapi.ImageSubscription,
					*image.Credentials,
				) ([]image.Image, error) {
					return nil, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Image: &kargoapi.ImageSubscription{
					RepoURL: "fake-repo",
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.ImageDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "fake-repo",
					},
				}, results)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := testCase.reconciler.discoverImages(
				context.TODO(),
				"fake-namespace",
				testCase.subs,
			)
			testCase.assertions(t, results, err)
		})
	}
}

func TestGetImageSourceURL(t *testing.T) {
	const testURLPrefix = "fake-url-prefix"
	testCases := []struct {
		name        string
		reconciler  *reconciler
		expectedURL string
	}{
		{
			name:        "no image source URL function found",
			reconciler:  &reconciler{},
			expectedURL: "",
		},
		{
			name: "image source URL function found",
			reconciler: &reconciler{
				imageSourceURLFnsByBaseURL: map[string]func(string, string) string{
					testURLPrefix: func(string, string) string {
						return "fake-url"
					},
				},
			},
			expectedURL: "fake-url",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedURL,
				testCase.reconciler.getImageSourceURL(testURLPrefix, "fake-tag"),
			)
		})
	}
}

func TestGetGithubImageSourceURL(t *testing.T) {
	const testTag = "fake-tag"
	testCases := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "with .git suffix",
			baseURL: "https://github.com/akuity/kargo.git",
		},
		{
			name:    "without .git suffix",
			baseURL: "https://github.com/akuity/kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				fmt.Sprintf(
					"%s/tree/%s",
					"https://github.com/akuity/kargo",
					testTag,
				),
				getGithubImageSourceURL(testCase.baseURL, testTag),
			)
		})
	}
}
