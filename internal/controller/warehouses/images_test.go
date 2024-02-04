package warehouses

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/image"
)

func TestSelectImages(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func([]kargoapi.Image, error)
	}{
		{
			name: "error getting latest version of an image",
			reconciler: &reconciler{
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
				getImageRefsFn: func(
					context.Context,
					string,
					kargoapi.ImageSelectionStrategy,
					string,
					string,
					[]string,
					string,
					*image.Credentials,
				) (string, string, error) {
					return "", "", errors.New("something went wrong")
				},
			},
			assertions: func(_ []kargoapi.Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting latest suitable image",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			reconciler: &reconciler{
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
				getImageRefsFn: func(
					context.Context,
					string,
					kargoapi.ImageSelectionStrategy,
					string,
					string,
					[]string,
					string,
					*image.Credentials,
				) (string, string, error) {
					return "fake-tag", "fake-digest", nil
				},
			},
			assertions: func(images []kargoapi.Image, err error) {
				require.NoError(t, err)
				require.Len(t, images, 1)
				require.Equal(
					t,
					kargoapi.Image{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
						Digest:  "fake-digest",
					},
					images[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.selectImages(
					context.Background(),
					"fake-namespace",
					[]kargoapi.RepoSubscription{
						{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-url",
							},
						},
					},
				),
			)
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
