package yaml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuoteIfNecessary(t *testing.T) {
	tests := []struct {
		input    any
		expected any
	}{
		{123, 123},
		{123.0, 123.0},
		{true, true},
		{false, false},
		{nil, nil},
		{"123", `"123"`},
		{"true", `"true"`},
		{"false", `"false"`},
		{"null", `"null"`},
		{"[1, 2, 3]", `"[1, 2, 3]"`},
		{`{"key": "value"}`, `"{\"key\": \"value\"}"`},
		{"string", "string"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			require.Equal(t, test.expected, QuoteIfNecessary(test.input))
		})
	}
}
