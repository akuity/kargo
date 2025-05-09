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
	for _, test := range []struct {
		name         string
		reader       io.Reader
		expectedCode int
	}{
		{
			name:         "ok",
			reader:       strings.NewReader("hello there"),
			expectedCode: http.StatusOK,
		},
		{
			name: "exceeds max",
			reader: func() io.Reader {
				b := make([]byte, maxBytes+1)
				return bytes.NewBuffer(b)
			}(),
			expectedCode: http.StatusRequestEntityTooLarge,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LimitRead(test.reader)
			receivedCode := http.StatusOK
			if err != nil {
				apiErr, ok := err.(*httpError)
				require.True(t, ok)
				receivedCode = apiErr.code
			}
			require.Equal(t,
				test.expectedCode,
				receivedCode,
			)
		})
	}
}
