package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequireFeature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	errDisabled := errors.New("disabled")
	testCases := []struct {
		name    string
		enabled bool
		assert  func(*testing.T, *httptest.ResponseRecorder, []*gin.Error, bool)
	}{
		{
			name: "disabled",
			assert: func(
				t *testing.T,
				w *httptest.ResponseRecorder,
				errs []*gin.Error,
				reached bool,
			) {
				require.False(t, reached)
				require.Len(t, errs, 1)
				require.ErrorIs(t, errs[0].Err, errDisabled)
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:    "enabled",
			enabled: true,
			assert: func(
				t *testing.T,
				w *httptest.ResponseRecorder,
				errs []*gin.Error,
				reached bool,
			) {
				require.True(t, reached)
				require.Empty(t, errs)
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := gin.New()
			// Stand in for the error-handling middleware: answer a reported
			// error with a status and keep the errors for the assertions.
			var reported []*gin.Error
			router.Use(func(c *gin.Context) {
				c.Next()
				reported = c.Errors
				if len(c.Errors) > 0 {
					c.Status(http.StatusNotImplemented)
				}
			})
			reached := false
			router.GET(
				"/",
				RequireFeature(testCase.enabled, errDisabled),
				func(c *gin.Context) {
					reached = true
					c.Status(http.StatusOK)
				},
			)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			testCase.assert(t, w, reported, reached)
		})
	}
}
