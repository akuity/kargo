package image

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewNewestBuildSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building tag based selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building tag based selector")
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
				l, ok := s.(*newestBuildSelector)
				require.True(t, ok)
				require.NotNil(t, l.tagBasedSelector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newNewestBuildSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_newestBuildSelector_sort(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	now := time.Now().UTC()

	images := []image{
		{CreatedAt: &now},
		{CreatedAt: timePtr(now.Add(time.Hour))},
		{CreatedAt: timePtr(now.Add(5 * time.Hour))},
		{CreatedAt: timePtr(now.Add(24 * time.Hour))},
		{CreatedAt: timePtr(now.Add(8 * time.Hour))},
		{CreatedAt: timePtr(now.Add(2 * time.Hour))},
		{CreatedAt: timePtr(now.Add(7 * time.Hour))},
		{CreatedAt: timePtr(now.Add(3 * time.Hour))},
	}

	(&newestBuildSelector{}).sort(images)

	require.Equal(
		t,
		[]image{
			{CreatedAt: timePtr(now.Add(24 * time.Hour))},
			{CreatedAt: timePtr(now.Add(8 * time.Hour))},
			{CreatedAt: timePtr(now.Add(7 * time.Hour))},
			{CreatedAt: timePtr(now.Add(5 * time.Hour))},
			{CreatedAt: timePtr(now.Add(3 * time.Hour))},
			{CreatedAt: timePtr(now.Add(2 * time.Hour))},
			{CreatedAt: timePtr(now.Add(time.Hour))},
			{CreatedAt: &now},
		},
		images,
	)
}

func Test_newestBuildSelector_Select(t *testing.T) {
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	s, err := newNewestBuildSelector(
		kargoapi.ImageSubscription{
			RepoURL:          "example/image",
			ExpressionFilter: `annotations["approved"] == "true"`,
		},
		nil,
	)
	require.NoError(t, err)
	n, ok := s.(*newestBuildSelector)
	require.True(t, ok)
	withFakeImages(
		n.repoClient,
		image{
			Tag:         "approved",
			CreatedAt:   &older,
			Annotations: map[string]string{"approved": "true"},
		},
		image{Tag: "not-approved", CreatedAt: &newer},
	)
	imgs, err := s.Select(t.Context())
	require.NoError(t, err)
	require.Len(t, imgs, 1)
	require.Equal(t, "approved", imgs[0].Tag)
}
