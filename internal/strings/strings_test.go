package strings

import (
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
