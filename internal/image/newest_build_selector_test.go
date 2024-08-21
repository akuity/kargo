package image

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewNewestBuildSelector(t *testing.T) {
	testOpts := SelectorOptions{
		AllowRegex:     "fake-regex",
		Ignore:         []string{"fake-ignore"},
		Platform:       "linux/amd64",
		DiscoveryLimit: 10,
	}
	s := newNewestBuildSelector(nil, testOpts)
	selector, ok := s.(*newestBuildSelector)
	require.True(t, ok)
	require.Equal(t, testOpts, selector.opts)
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
