package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:  "string",
			input: "string-error",
			expected: `{"error":"string-error"}
`,
		},
		{
			name:  "error",
			input: errors.New("standard-error"),
			expected: `{"error":"standard-error"}
`,
		},
		{
			name: "unexpected type",
			input: map[string]string{
				"error-1": "msg-1",
				"error-2": "msg-2",
			},
			expected: `{"error":"map[error-1:msg-1 error-2:msg-2]"}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, http.StatusInternalServerError, test.input)
			require.Equal(t,
				http.StatusInternalServerError,
				w.Result().StatusCode,
			)
			require.Equal(t,
				test.expected,
				w.Body.String(),
			)
		})
	}
}

func TestErrorf(t *testing.T) {
	w := httptest.NewRecorder()
	Errorf(w, http.StatusInternalServerError,
		"this error has occurred %d time",
		1,
	)
	require.Equal(t,
		http.StatusInternalServerError,
		w.Result().StatusCode,
	)
	require.Equal(t,
		`{"error":"this error has occurred 1 time"}
`, w.Body.String(),
	)
}
