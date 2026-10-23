package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	libhttp "github.com/akuity/kargo/pkg/http"
)

func TestBindJSON(t *testing.T) {
	t.Parallel()
	type body struct {
		Name string `json:"name"`
	}
	testCases := []struct {
		name   string
		body   string
		assert func(*testing.T, body, bool, *gin.Context)
	}{
		{
			name: "well-formed",
			body: `{"name": "x"}`,
			assert: func(t *testing.T, b body, ok bool, c *gin.Context) {
				require.True(t, ok)
				require.Equal(t, "x", b.Name)
				require.Empty(t, c.Errors)
			},
		},
		{
			name: "malformed",
			body: `{`,
			assert: func(t *testing.T, _ body, ok bool, c *gin.Context) {
				require.False(t, ok)
				require.Len(t, c.Errors, 1)
				var httpErr *libhttp.HTTPError
				require.ErrorAs(t, c.Errors[0].Err, &httpErr)
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testCase.body))
			var b body
			ok := BindJSON(c, &b)
			testCase.assert(t, b, ok, c)
		})
	}
}
