package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeFrom(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    func() error
		expected int
	}{
		{
			name: "bad request",
			input: func() error {
				return BadRequestError("malformed request")
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "bad request formatted",
			input: func() error {
				return BadRequestErrorf(
					"%d is invalid",
					-1,
				)
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "unauthorized",
			input: func() error {
				return UnauthorizedError("token expired")
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "unauthorized formatted",
			input: func() error {
				return UnauthorizedErrorf(
					"reason: %s",
					"device not recognized",
				)
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "forbidden",
			input: func() error {
				return ForbiddenError("action not allowed")
			},
			expected: http.StatusForbidden,
		},
		{
			name: "forbidden formatted",
			input: func() error {
				return ForbiddenErrorf(
					"requires %s permissions",
					"admin",
				)
			},
			expected: http.StatusForbidden,
		},
		{
			name: "internal server error",
			input: func() error {
				return ServerError("some vague error obscuring db info")
			},
			expected: http.StatusInternalServerError,
		},
		{
			name: "internal server error formatted",
			input: func() error {
				return ServerErrorf(
					"max retries(%d) exceeded",
					3,
				)
			},
			expected: http.StatusInternalServerError,
		},
		{
			name: "unimplemented",
			input: func() error {
				return UnimplementedError("not implemented")
			},
			expected: http.StatusNotImplemented,
		},
		{
			name: "unimplemented - formatted",
			input: func() error {
				return UnimplementedErrorf(
					"not implemented, use %s instead",
					"other-endpoint",
				)
			},
			expected: http.StatusNotImplemented,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t,
				test.expected,
				CodeFrom(test.input()),
			)
		})
	}
}

func TestStatusErrors(t *testing.T) {
	for _, test := range []struct {
		name         string
		writeFn      func() *httptest.ResponseRecorder
		expectedMsg  string
		expectedCode int
	}{
		{
			name: "bad request",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteBadRequestError(w, "malformed payload")
				return w
			},
			expectedMsg:  "{\"error\":\"Bad Request: malformed payload\"}\n",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "bad request - formatted",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteBadRequestErrorf(w,
					"%q is an invalid url",
					"myinvalidurl",
				)
				return w
			},
			expectedMsg:  "{\"error\":\"Bad Request: \\\"myinvalidurl\\\" is an invalid url\"}\n",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "unauthorized",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteUnauthorizedError(w, "token expired")
				return w
			},
			expectedMsg:  "{\"error\":\"Unauthorized: token expired\"}\n",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "unauthorized - formatted",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteUnauthorizedErrorf(w,
					"reason: %s",
					"device not recognized",
				)
				return w
			},
			expectedMsg:  "{\"error\":\"Unauthorized: reason: device not recognized\"}\n",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "forbidden",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteForbiddenError(w, "action not allowed")
				return w
			},
			expectedMsg:  "{\"error\":\"Forbidden: action not allowed\"}\n",
			expectedCode: http.StatusForbidden,
		},
		{
			name: "forbidden - formatted",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteForbiddenErrorf(w,
					"requires %s permissions",
					"admin",
				)
				return w
			},
			expectedMsg:  "{\"error\":\"Forbidden: requires admin permissions\"}\n",
			expectedCode: http.StatusForbidden,
		},
		{
			name: "internal server error",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteServerError(w, "some vague error obscuring db info")
				return w
			},
			expectedMsg:  "{\"error\":\"Internal Server Error: some vague error obscuring db info\"}\n",
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "internal server error - formatted",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteServerErrorf(w,
					"max retries(%d) exceeded",
					3,
				)
				return w
			},
			expectedMsg:  "{\"error\":\"Internal Server Error: max retries(3) exceeded\"}\n",
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "unimplemented",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteUnimplementedError(w, "not implemented")
				return w
			},
			expectedMsg:  "",
			expectedCode: http.StatusNotImplemented,
		},
		{
			name: "unimplemented - formatted",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteUnimplementedErrorf(w,
					"not implemented see %s",
					"other thing",
				)
				return w
			},
			expectedMsg:  "",
			expectedCode: http.StatusNotImplemented,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := test.writeFn()
			require.Equal(t,
				test.expectedCode,
				w.Result().StatusCode,
			)
			require.Equal(t,
				test.expectedMsg,
				w.Body.String(),
			)
		})
	}
}

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
			WriteError(w, http.StatusInternalServerError, test.input)
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
	WriteErrorf(w, http.StatusInternalServerError,
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
