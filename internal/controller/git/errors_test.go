package git

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMergeConflict(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not a a merge conflict",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "a merge conflict",
			err:      ErrMergeConflict,
			expected: true,
		},
		{
			name:     "a wrapped merge conflict",
			err:      fmt.Errorf("an error occurred: %w", ErrMergeConflict),
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := IsMergeConflict(testCase.err)
			require.Equal(t, testCase.expected, actual)
		})
	}
}
