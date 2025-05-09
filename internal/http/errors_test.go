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
		name         string
		writeFn      func() *httptest.ResponseRecorder
		expectedCode int
		expectedBody string
	}{
		{
			name: "basic error",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteErrorJSON(w, errors.New("basic error"))
				return w
			},
			// should default to this
			expectedCode: http.StatusInternalServerError,
			expectedBody: "{\"error\":\"basic error\"}\n",
		},
		{
			name: "http error",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteErrorJSON(w,
					Error(
						errors.New("unauthorized"),
						http.StatusUnauthorized,
					),
				)
				return w
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "{\"error\":\"unauthorized\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := test.writeFn()
			require.Equal(t, test.expectedCode, w.Result().StatusCode)
			require.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}

func TestWriteResponseJSON(t *testing.T) {
	for _, test := range []struct {
		name         string
		writeFn      func() *httptest.ResponseRecorder
		expectedCode int
		expectedBody string
	}{
		{
			name: "nil body",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteResponseJSON(w,
					http.StatusOK,
					nil,
				)
				return w
			},
			expectedCode: http.StatusOK,
			expectedBody: "{}\n",
		},
		{
			name: "non-nil body",
			writeFn: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				WriteResponseJSON(w,
					http.StatusOK,
					map[string]any{
						"key": "value",
					},
				)
				return w
			},
			expectedCode: http.StatusOK,
			expectedBody: "{\"key\":\"value\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := test.writeFn()
			require.Equal(t, test.expectedCode, w.Result().StatusCode)
			require.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}
