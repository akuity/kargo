package image

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewNewestBuildSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	s := newNewestBuildSelector(nil, testAllowRegex, testIgnore, testPlatform)
	selector, ok := s.(*newestBuildSelector)
	require.True(t, ok)
	require.Equal(t, testAllowRegex, selector.allowRegex)
	require.Equal(t, testIgnore, selector.ignore)
	require.Equal(t, testPlatform, selector.platform)
}

func TestSortImagesByDate(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	now := time.Now().UTC()

	images := []Image{
		{CreatedAt: &now},
		{CreatedAt: timePtr(now.Add(time.Hour))},
		{CreatedAt: timePtr(now.Add(5 * time.Hour))},
		{CreatedAt: timePtr(now.Add(24 * time.Hour))},
		{CreatedAt: timePtr(now.Add(8 * time.Hour))},
		{CreatedAt: timePtr(now.Add(2 * time.Hour))},
		{CreatedAt: timePtr(now.Add(7 * time.Hour))},
		{CreatedAt: timePtr(now.Add(3 * time.Hour))},
	}

	sortImagesByDate(images)

	require.Equal(
		t,
		[]Image{
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
