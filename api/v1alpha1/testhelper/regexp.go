package testhelper

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// ValidateRegularExpression ensures that the provided regular expression
// matches the expected results for the given test cases. Each key in the
// testCases is tested against the expression, and the result must match the
// value in the map of testcases.
func ValidateRegularExpression(t *testing.T, expression *regexp.Regexp, testCases map[string]bool) {
	for tt, expected := range testCases {
		t.Run(tt, func(t *testing.T) {
			require.Equal(t, expected, expression.MatchString(tt))
		})
	}
}
