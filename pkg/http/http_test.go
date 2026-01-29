package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteResponseJSON(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   any
		code    int
		bodyObj any
	}{
		{
			name:    "nil body",
			input:   nil,
			code:    http.StatusOK,
			bodyObj: "{}\n",
		},
		{
			name:    "non-nil body",
			code:    http.StatusOK,
			input:   map[string]string{"key": "value"},
			bodyObj: "{\"key\":\"value\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteResponseJSON(w, test.code, test.input)
			require.Equal(t, test.code, w.Result().StatusCode)
			require.Equal(t, test.bodyObj, w.Body.String())
		})
	}
}

func TestSetCacheHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SetCacheHeaders(w, 30*24*time.Hour, 7*24*time.Hour)
	headers := w.Result().Header
	require.Equal(t, "public, max-age=2592000", headers.Get("Cache-Control"))
	expires := headers.Get("Expires")
	require.Equal(t, expires[len(expires)-4:], " GMT", "Expires header must end with ' GMT', got '%s'.", expires)
}
