package image

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewNewestBuildTagSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	s := newNewestBuildTagSelector(nil, testAllowRegex, testIgnore, testPlatform)
	selector, ok := s.(*newestBuildTagSelector)
	require.True(t, ok)
	require.Equal(t, testAllowRegex, selector.allowRegex)
	require.Equal(t, testIgnore, selector.ignore)
	require.Equal(t, testPlatform, selector.platform)
}

func TestSortTagsByDate(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	now := time.Now().UTC()

	tags := []Tag{
		{CreatedAt: &now},
		{CreatedAt: timePtr(now.Add(time.Hour))},
		{CreatedAt: timePtr(now.Add(5 * time.Hour))},
		{CreatedAt: timePtr(now.Add(24 * time.Hour))},
		{CreatedAt: timePtr(now.Add(8 * time.Hour))},
		{CreatedAt: timePtr(now.Add(2 * time.Hour))},
		{CreatedAt: timePtr(now.Add(7 * time.Hour))},
		{CreatedAt: timePtr(now.Add(3 * time.Hour))},
	}

	sortTagsByDate(tags)

	require.Equal(
		t,
		[]Tag{
			{CreatedAt: timePtr(now.Add(24 * time.Hour))},
			{CreatedAt: timePtr(now.Add(8 * time.Hour))},
			{CreatedAt: timePtr(now.Add(7 * time.Hour))},
			{CreatedAt: timePtr(now.Add(5 * time.Hour))},
			{CreatedAt: timePtr(now.Add(3 * time.Hour))},
			{CreatedAt: timePtr(now.Add(2 * time.Hour))},
			{CreatedAt: timePtr(now.Add(time.Hour))},
			{CreatedAt: &now},
		},
		tags,
	)
}
