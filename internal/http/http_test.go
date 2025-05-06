package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeakBody(t *testing.T) {
	expected := "hello there"
	req, err := http.NewRequest(
		http.MethodGet,
		"https://doesntmatter.com",
		strings.NewReader(expected),
	)
	require.NoError(t, err)

	for range 5 {
		b, err := PeakBody(req)
		require.NoError(t, err)
		require.Equal(t, expected, string(b))
	}
}
