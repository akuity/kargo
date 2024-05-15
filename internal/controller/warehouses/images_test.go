package warehouses

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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
