package http

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitRead(t *testing.T) {
	const maxBytes = 2 << 20 // 2MB
	for _, test := range []struct {
		name   string
		reader io.ReadCloser
		code   int
	}{
		{
			name:   "ok",
			reader: io.NopCloser(strings.NewReader("hello there")),
			code:   http.StatusOK,
		},
		{
			name: "exceeds max",
			reader: func() io.ReadCloser {
				b := make([]byte, maxBytes+1)
				return io.NopCloser(bytes.NewBuffer(b))
			}(),
			code: http.StatusRequestEntityTooLarge,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LimitRead(test.reader, maxBytes)
			receivedCode := http.StatusOK
			if err != nil {
				apiErr, ok := err.(*httpError)
				require.True(t, ok)
				receivedCode = apiErr.code
			}
			require.Equal(t, test.code, receivedCode)
		})
	}
}
