package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBind(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		bind   Binder[string]
		status int
		called bool
	}{
		{
			name:   "a bound value reaches the handler",
			bind:   func(*gin.Context) (string, bool) { return "value", true },
			status: http.StatusOK,
			called: true,
		},
		{
			name: "a request that does not bind never reaches the handler",
			bind: func(c *gin.Context) (string, bool) {
				c.AbortWithStatus(http.StatusBadRequest)
				return "", false
			},
			status: http.StatusBadRequest,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got string
			called := false
			router := gin.New()
			router.POST("/", Bind(testCase.bind, func(c *gin.Context, v string) {
				called = true
				got = v
				c.Status(http.StatusOK)
			}))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
			require.Equal(t, testCase.status, w.Code)
			require.Equal(t, testCase.called, called)
			if called {
				require.Equal(t, "value", got)
			}
		})
	}
}
