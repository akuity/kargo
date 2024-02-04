package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLexicalSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	s := newLexicalSelector(nil, testAllowRegex, testIgnore, testPlatform)
	selector, ok := s.(*lexicalSelector)
	require.True(t, ok)
	require.Equal(t, testAllowRegex, selector.allowRegex)
	require.Equal(t, testIgnore, selector.ignore)
	require.Equal(t, testPlatform, selector.platform)
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
