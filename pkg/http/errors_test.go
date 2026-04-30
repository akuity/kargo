package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteErrorJSON(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		code    int
		bodyObj any
	}{
		{
			name:    "server error that leaks db info",
			err:     errors.New("pg error code abc123"),
			code:    http.StatusInternalServerError,
			bodyObj: "{}\n",
		},
		{
			name: "http error",
			err: Error(
				errors.New("unauthorized"),
				http.StatusUnauthorized,
			),
			code:    http.StatusUnauthorized,
			bodyObj: "{\"error\":\"unauthorized\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteErrorJSON(w, test.err)
			require.Equal(t, test.code, w.Result().StatusCode)
			require.Equal(t, test.bodyObj, w.Body.String())
		})
	}
}
