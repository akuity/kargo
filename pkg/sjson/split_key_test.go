package sjson

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitKey(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		expected    []string
		errContains string
	}{
		{
			name:        "empty key",
			errContains: "empty key",
		},
		{
			name:        "starts with dot",
			key:         ".foo",
			errContains: "empty key part in key",
		},
		{
			name:        "ends with dot",
			key:         "foo.",
			errContains: "empty key part in key",
		},
		{
			name:        "double dots",
			key:         "foo..bar",
			errContains: "empty key part in key",
		},
		{
			name:        "invalid escape sequence",
			key:         `foo\nbar`,
			errContains: "invalid escape sequence",
		},
		{
			name:        "invalid use of colon",
			key:         `foo:bar`,
			errContains: "unexpected colon in key",
		},
		{
			name:     "basic split",
			key:      "foo.bar.bat.baz",
			expected: []string{"foo", "bar", "bat", "baz"},
		},
		{
			name:     "split key with escaped dots",
			key:      `foo\.bar.bat\.baz`,
			expected: []string{"foo.bar", "bat.baz"},
		},
		{
			name:     "split key with escaped colon",
			key:      `foo.bar\:bat.baz`,
			expected: []string{"foo", "bar:bat", "baz"},
		},
		{
			name:     "split key with unescaped colon",
			key:      `foo.bar.:2.baz`,
			expected: []string{"foo", "bar", "2", "baz"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parts, err := SplitKey(testCase.key)
			if testCase.errContains != "" {
				require.ErrorContains(t, err, testCase.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, parts)
			}
		})
	}
}
