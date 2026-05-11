package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBodyLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		maxBytes       int64
		bodySize       int
		expectedStatus int
		expectBodyRead bool
	}{
		{
			name:           "POST body under limit",
			method:         http.MethodPost,
			maxBytes:       1024,
			bodySize:       512,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "POST body exactly at limit",
			method:         http.MethodPost,
			maxBytes:       1024,
			bodySize:       1024,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "POST body over limit",
			method:         http.MethodPost,
			maxBytes:       1024,
			bodySize:       2048,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectBodyRead: false,
		},
		{
			name:           "POST empty body",
			method:         http.MethodPost,
			maxBytes:       1024,
			bodySize:       0,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "PUT body over limit",
			method:         http.MethodPut,
			maxBytes:       100,
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectBodyRead: false,
		},
		{
			name:           "PUT body under limit",
			method:         http.MethodPut,
			maxBytes:       100,
			bodySize:       50,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "PATCH body over limit",
			method:         http.MethodPatch,
			maxBytes:       100,
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectBodyRead: false,
		},
		{
			name:           "PATCH body under limit",
			method:         http.MethodPatch,
			maxBytes:       100,
			bodySize:       50,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "DELETE body over limit",
			method:         http.MethodDelete,
			maxBytes:       100,
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectBodyRead: false,
		},
		{
			name:           "DELETE body under limit",
			method:         http.MethodDelete,
			maxBytes:       100,
			bodySize:       50,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
		{
			name:           "GET request not affected",
			method:         http.MethodGet,
			maxBytes:       10,
			bodySize:       0,
			expectedStatus: http.StatusOK,
			expectBodyRead: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			// Error handling middleware (mimics handleError behavior)
			router.Use(func(c *gin.Context) {
				c.Next()
				if len(c.Errors) > 0 {
					if _, ok := c.Errors.Last().Err.(*http.MaxBytesError); ok {
						c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
						return
					}
				}
			})

			router.Use(bodyLimitMiddleware(tt.maxBytes))

			var bodyReadSuccessfully bool
			router.Handle(tt.method, "/test", func(c *gin.Context) {
				_, err := io.ReadAll(c.Request.Body)
				if err != nil {
					_ = c.Error(err)
					return
				}
				bodyReadSuccessfully = true
				c.Status(http.StatusOK)
			})

			var body io.Reader
			if tt.bodySize > 0 {
				body = bytes.NewBufferString(strings.Repeat("x", tt.bodySize))
			}
			req := httptest.NewRequest(tt.method, "/test", body)
			if tt.bodySize > 0 {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			require.Equal(t, tt.expectBodyRead, bodyReadSuccessfully)
		})
	}
}
