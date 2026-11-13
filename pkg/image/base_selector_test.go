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
			name: "error compiling filter expression",
			sub: kargoapi.ImageSubscription{
				RepoURL:          "example/image",
				ExpressionFilter: "(1 + 2",
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error compiling filter expression")
			},
		},
		{
			name: "unknown variable in filter expression",
			sub: kargoapi.ImageSubscription{
				RepoURL:          "example/image",
				ExpressionFilter: "annotation == 'approved'",
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error compiling filter expression")
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

func Test_matchesFilterExpression(t *testing.T) {
	createdAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	img := image{
		Tag:         "v1.0.0",
		Digest:      "sha256:abc",
		CreatedAt:   &createdAt,
		Annotations: map[string]string{"approved": "true"},
	}
	testCases := []struct {
		name       string
		expression string
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "no expression",
			assertions: func(t *testing.T, matched bool, err error) {
				require.NoError(t, err)
				require.True(t, matched)
			},
		},
		{
			name: "expression uses all variables",
			expression: `tag == "v1.0.0" && digest == "sha256:abc" && ` +
				`annotations["approved"] == "true" && createdAt.Year() == 2025`,
			assertions: func(t *testing.T, matched bool, err error) {
				require.NoError(t, err)
				require.True(t, matched)
			},
		},
		{
			name:       "expression not satisfied",
			expression: `hasPrefix(tag, "v2.")`,
			assertions: func(t *testing.T, matched bool, err error) {
				require.NoError(t, err)
				require.False(t, matched)
			},
		},
		{
			name:       "non-boolean result converted to boolean",
			expression: "0",
			assertions: func(t *testing.T, matched bool, err error) {
				require.NoError(t, err)
				require.False(t, matched)
			},
		},
		{
			name:       "non-boolean result not convertible to boolean",
			expression: `"foo"`,
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "error evaluating filter expression")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newBaseSelector(
				kargoapi.ImageSubscription{
					RepoURL:          "example/image",
					ExpressionFilter: testCase.expression,
				},
				nil,
				false,
			)
			require.NoError(t, err)
			matched, err := matchesFilterExpression(s.filterExpression, img, nil)
			testCase.assertions(t, matched, err)
		})
	}
}
