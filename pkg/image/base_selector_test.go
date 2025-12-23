package image

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewBaseSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, *baseSelector, error)
	}{
		{
			name: "error parsing platform constraint",
			sub:  kargoapi.ImageSubscription{Platform: "invalid"},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error parsing platform constraint")
			},
		},
		{
			name: "error creating repository client",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(
					t,
					err,
					"error creating repository client for image",
				)
			},
		},
		{
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL:  "example/image",
				Platform: "linux/amd64",
			},
			assertions: func(t *testing.T, s *baseSelector, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&platformConstraint{
						os:   "linux",
						arch: "amd64",
					},
					s.platformConstraint,
				)
				require.NotNil(t, s.repoClient)
				require.True(t, s.repoClient.cacheByTag)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newBaseSelector(testCase.sub, nil, true)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_baseSelector_imagesToAPIImages(t *testing.T) {
	now := time.Now()
	apiImages := (&baseSelector{}).imagesToAPIImages(
		[]image{
			{
				Tag:         "foo",
				Digest:      "foo-digest",
				Annotations: map[string]string{"my-annotation": "foo"},
				CreatedAt:   &now,
			},
			{
				Tag:         "bar",
				Digest:      "bar-digest",
				Annotations: map[string]string{"my-annotation": "bar"},
				CreatedAt:   &now,
			},
			{
				Tag:         "bat",
				Digest:      "bat-digest",
				Annotations: map[string]string{"my-annotation": "bat"},
				CreatedAt:   &now,
			},
		},
		2, // Limit
	)
	require.Equal(
		t,
		[]kargoapi.DiscoveredImageReference{
			{
				Tag:         "foo",
				Digest:      "foo-digest",
				Annotations: map[string]string{"my-annotation": "foo"},
				CreatedAt:   &v1.Time{Time: now},
			},
			{
				Tag:         "bar",
				Digest:      "bar-digest",
				Annotations: map[string]string{"my-annotation": "bar"},
				CreatedAt:   &v1.Time{Time: now},
			},
		},
		apiImages,
	)
}
