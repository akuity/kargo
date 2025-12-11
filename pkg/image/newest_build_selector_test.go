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
