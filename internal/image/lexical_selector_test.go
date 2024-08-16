package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLexicalSelector(t *testing.T) {
	testOpts := SelectorOptions{
		AllowRegex:     "fake-regex",
		Ignore:         []string{"fake-ignore"},
		Platform:       "linux/amd64",
		DiscoveryLimit: 10,
	}
	s := newLexicalSelector(nil, testOpts)
	selector, ok := s.(*lexicalSelector)
	require.True(t, ok)
	require.Equal(t, testOpts, selector.opts)
}

func TestSortTagsLexically(t *testing.T) {
	tags := []string{"a", "z", "b", "y", "c", "x", "d", "w", "e", "v"}
	sortTagsLexically(tags)
	require.Equal(
		t,
		[]string{"z", "y", "x", "w", "v", "e", "d", "c", "b", "a"},
		tags,
	)
}
