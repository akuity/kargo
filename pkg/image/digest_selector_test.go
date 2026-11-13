package image

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewDigestSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building base selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL:    "example/image",
				Constraint: "latest",
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				d, ok := s.(*digestSelector)
				require.True(t, ok)
				require.Equal(t, "latest", d.mutableTag)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newDigestSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_digestSelector_Select(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		assertions  func(*testing.T, []kargoapi.DiscoveredImageReference, error)
	}{
		{
			name:        "image satisfies filter expression",
			annotations: map[string]string{"approved": "true"},
			assertions: func(t *testing.T, imgs []kargoapi.DiscoveredImageReference, err error) {
				require.NoError(t, err)
				require.Len(t, imgs, 1)
			},
		},
		{
			name: "image does not satisfy filter expression",
			assertions: func(t *testing.T, imgs []kargoapi.DiscoveredImageReference, err error) {
				require.NoError(t, err)
				require.Empty(t, imgs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newDigestSelector(
				kargoapi.ImageSubscription{
					RepoURL:          "example/image",
					Constraint:       "latest",
					ExpressionFilter: `annotations["approved"] == "true"`,
				},
				nil,
			)
			require.NoError(t, err)
			d, ok := s.(*digestSelector)
			require.True(t, ok)
			withFakeImages(
				d.repoClient,
				image{Tag: "latest", Annotations: testCase.annotations},
			)
			imgs, err := s.Select(t.Context())
			testCase.assertions(t, imgs, err)
		})
	}
}
