package strings

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitLast(t *testing.T) {
	testCases := []struct {
		name          string
		s             string
		sep           string
		expectError   bool
		expectedLeft  string
		expectedRight string
	}{
		{
			name:          "simple case",
			s:             "foo:bar:bat",
			sep:           ":",
			expectError:   false,
			expectedLeft:  "foo:bar",
			expectedRight: "bat",
		},
		{
			name:          "string starts with last occurrence of separator",
			s:             ":foo",
			sep:           ":",
			expectError:   false,
			expectedLeft:  "",
			expectedRight: "foo",
		},
		{
			name:          "string ends with last occurrence of separator",
			s:             "foo:bar:bat:",
			sep:           ":",
			expectError:   false,
			expectedLeft:  "foo:bar:bat",
			expectedRight: "",
		},
		{
			name:        "string contains no occurrences of separator",
			s:           "foo:bar:bat:",
			sep:         "!",
			expectError: true,
		},
		{
			// This is a special case of the string not containing the separator
			name:        "string is empty",
			s:           "",
			sep:         ":",
			expectError: true,
		},
		{
			name:        "no separator specified",
			s:           "foo:bar:bat:",
			sep:         "",
			expectError: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			left, right, err := SplitLast(testCase.s, testCase.sep)
			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedLeft, left)
				require.Equal(t, testCase.expectedRight, right)
			}
		})
	}
}

func TestHashShorten(t *testing.T) {
	const in = "Four score and seven years ago"
	const sep = "... "
	shortHashLen := defaultShortHashLen

	t.Run("input is shorter than requested max length", func(t *testing.T) {
		out, nowShort := HashShorten(in, len(in), sep, 0)
		require.True(t, nowShort)
		require.Equal(t, in, out)
	})

	t.Run("requested max length is shorter than short hash length", func(t *testing.T) {
		out, nowShort := HashShorten(in, shortHashLen-1, sep, 0)
		require.False(t, nowShort)
		require.Equal(t, in, out)
	})

	t.Run("shortened to the exact length of the short hash", func(t *testing.T) {
		out, nowShort := HashShorten(in, shortHashLen, sep, 0)
		require.True(t, nowShort)
		// The result should be the short hash only
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(in)))
		require.Equal(t, out, sum[:shortHashLen])
	})

	t.Run("shortened without enough room for original chars + separator", func(t *testing.T) {
		out, nowShort := HashShorten(in, len(sep)+shortHashLen, sep, 0)
		require.True(t, nowShort)
		// The result should be the short hash only
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(in)))
		require.Equal(t, out, sum[:shortHashLen])
	})

	t.Run("shortened with enough room for original chars + separator", func(t *testing.T) {
		maxLength := 1 + len(sep) + shortHashLen
		out, nowShort := HashShorten(in, maxLength, sep, 0)
		require.True(t, nowShort)
		// The result should be the short hash only
		require.Len(t, out, maxLength)
		// The result should contain the separator
		require.Contains(t, out, sep)
		// The trailing characters of the result should be the short hash
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(in)))
		require.Equal(t, out[len(out)-shortHashLen:], sum[:shortHashLen])
	})

	t.Run("no double separators in the result", func(t *testing.T) {
		// Carefully engineering input that will result in a double separator that
		// we expect to be removed from the final result...
		const firstPart = "Four score and seven"
		const secondPart = "years, ago our fathers brought forth, on this continent"
		in := firstPart + sep + secondPart
		maxLength := len(firstPart) + 2*len(sep) + shortHashLen
		out, nowShort := HashShorten(in, maxLength, sep, 0)
		require.True(t, nowShort)
		// The separator should be found only once in the result
		require.Equal(t, 1, strings.Count(out, sep))
		// The trailing characters of the result should be the short hash
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(in)))
		require.Equal(t, out[len(out)-shortHashLen:], sum[:shortHashLen])
	})

	t.Run("separator defaults to single dash", func(t *testing.T) {
		// maxLength is enough room for one original character, a dash, and the
		// short hash
		maxLength := 1 + 1 + shortHashLen
		out, nowShort := HashShorten(in, maxLength, "", 0)
		require.True(t, nowShort)
		require.Contains(t, out, "-")
		// The trailing characters of the result should be the short hash
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(in)))
		require.Equal(t, out[len(out)-shortHashLen:], sum[:shortHashLen])
	})

	t.Run("results are deterministic", func(t *testing.T) {
		maxLength := 1 + len(sep) + shortHashLen
		out1, nowShort := HashShorten(in, maxLength, sep, 0)
		require.True(t, nowShort)
		out2, nowShort := HashShorten(in, maxLength, sep, 0)
		require.True(t, nowShort)
		require.Equal(t, out1, out2)
	})

	t.Run("different input yields different results", func(t *testing.T) {
		maxLength := 1 + len(sep) + shortHashLen
		out1, nowShort := HashShorten(in, maxLength, sep, 0)
		require.True(t, nowShort)
		out2, nowShort := HashShorten(
			"Five score and seven years ago",
			maxLength,
			sep,
			0,
		)
		require.True(t, nowShort)
		require.NotEqual(t, out1, out2)
	})
}
