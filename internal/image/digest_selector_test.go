package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDigestSelector(t *testing.T) {
	testOpts := SelectorOptions{
		Constraint: "fake-constraint",
		platform: &platformConstraint{
			os:   "linux",
			arch: "amd64",
		},
	}
	s, err := newDigestSelector(nil, testOpts)
	require.NoError(t, err)
	selector, ok := s.(*digestSelector)
	require.True(t, ok)
	require.Equal(t, testOpts, selector.opts)
}
