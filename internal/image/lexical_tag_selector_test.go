package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLexicalTagSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	s := newLexicalTagSelector(nil, testAllowRegex, testIgnore, testPlatform)
	selector, ok := s.(*lexicalTagSelector)
	require.True(t, ok)
	require.Equal(t, testAllowRegex, selector.allowRegex)
	require.Equal(t, testIgnore, selector.ignore)
	require.Equal(t, testPlatform, selector.platform)
}

func TestSortTagsNamesLexically(t *testing.T) {
	tagNames := []string{"a", "z", "b", "y", "c", "x", "d", "w", "e", "v"}
	sortTagNamesLexically(tagNames)
	require.Equal(
		t,
		[]string{"z", "y", "x", "w", "v", "e", "d", "c", "b", "a"},
		tagNames,
	)
}
