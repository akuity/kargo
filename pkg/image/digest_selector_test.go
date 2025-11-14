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
