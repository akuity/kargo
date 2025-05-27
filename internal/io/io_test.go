package io

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitRead(t *testing.T) {
	const maxBytes = 2 << 20 // 2MB
	for _, test := range []struct {
		name   string
		reader io.ReadCloser
		errMsg string
	}{
		{
			name:   "ok",
			reader: io.NopCloser(strings.NewReader("hello there")),
			errMsg: "",
		},
		{
			name: "exceeds limit",
			reader: func() io.ReadCloser {
				b := make([]byte, maxBytes+1)
				return io.NopCloser(bytes.NewBuffer(b))
			}(),
			errMsg: "reader exceeds limit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LimitRead(test.reader, maxBytes)
			if test.errMsg == "" {
				require.NoError(t, err)
				return
			}
			require.Contains(t, err.Error(), test.errMsg)
		})
	}
}
