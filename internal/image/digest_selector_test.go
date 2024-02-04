package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDigestSelector(t *testing.T) {
	const testConstraint = "fake-constraint"
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	s, err := newDigestSelector(nil, testConstraint, testPlatform)
	require.NoError(t, err)
	selector, ok := s.(*digestSelector)
	require.True(t, ok)
	require.Equal(t, testConstraint, selector.constraint)
	require.Equal(t, testPlatform, selector.platform)
}
