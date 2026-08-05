package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		handler        gin.HandlerFunc
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no panic",
			handler:        func(c *gin.Context) { c.Status(http.StatusOK) },
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			// The panic must reach the error-handling middleware, which answers it
			// with the same body every other error gets, disclosing nothing.
			name:           "panic",
			handler:        func(*gin.Context) { panic("kaboom") },
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			name:           "panic with an error value",
			handler:        func(*gin.Context) { panic(errors.New("kaboom")) },
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			// There is no client left to answer, so nothing is written. Gin's
			// recorder reports the status it was initialized with.
			name:           "client went away",
			handler:        func(*gin.Context) { panic(syscall.EPIPE) },
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "handler aborted",
			handler:        func(*gin.Context) { panic(http.ErrAbortHandler) },
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			s := &server{}
			router := gin.New()
			router.Use(s.handleError)
			router.Use(recoveryMiddleware())
			router.GET("/", testCase.handler)

			w := httptest.NewRecorder()
			// A panic escaping the middleware would fail the test by crashing it.
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, testCase.expectedStatus, w.Code)
			require.Equal(t, testCase.expectedBody, w.Body.String())
		})
	}
}
