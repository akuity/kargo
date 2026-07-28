package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_normalizeListResourceVersion(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		rv       string
		expected string
	}{
		{name: "real rv passthrough", rv: "12345", expected: "12345"},
		{name: "zero sentinel becomes empty", rv: "0", expected: ""},
		{name: "empty stays empty", rv: "", expected: ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, normalizeListResourceVersion(tc.rv))
		})
	}
}
