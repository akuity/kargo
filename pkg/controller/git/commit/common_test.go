package commit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShortenString(t *testing.T) {
	testCases := []struct {
		name   string
		str    string
		length int
		want   string
	}{
		{
			name:   "exceeds length",
			str:    "this is a long string",
			length: 10,
			want:   "this is...",
		},
		{
			name:   "equal length",
			str:    "this is a long string",
			length: 21,
			want:   "this is a long string",
		},
		{
			name:   "shorter length",
			str:    "this is a long string",
			length: 30,
			want:   "this is a long string",
		},
		{
			name:   "empty string",
			str:    "",
			length: 10,
			want:   "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, shortenString(testCase.str, testCase.length))
		})
	}
}
