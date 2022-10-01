package dockerhub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTokenFilterConfig(t *testing.T) {
	config, ok := NewTokenFilterConfig().(*tokenFilterConfig)
	require.True(t, ok)
	require.NotNil(t, config.tokens)
}

func TestAddToken(t *testing.T) {
	const testToken = "foo"
	config, ok := NewTokenFilterConfig().(*tokenFilterConfig)
	require.True(t, ok)
	require.Empty(t, config.tokens)
	config.AddToken(testToken)
	require.Len(t, config.tokens, 1)
	require.Contains(t, config.tokens, testToken)
}

func TestHasToken(t *testing.T) {
	const testToken = "foo"
	config := tokenFilterConfig{
		tokens: map[string]struct{}{
			testToken: {},
		},
	}
	require.True(t, config.HasToken(testToken))
	require.False(t, config.HasToken("bogus"))
}

func TestTokenFilter(t *testing.T) {
	testConfig := NewTokenFilterConfig()
	const testToken = "bar"
	testConfig.AddToken(testToken)
	testCases := []struct {
		name       string
		setup      func() *http.Request
		assertions func(handlerCalled bool, rr *http.Response)
	}{
		{
			name: "valid token provided",
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				q := req.URL.Query()
				q.Set("access_token", testToken)
				req.URL.RawQuery = q.Encode()
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusOK, r.StatusCode)
				require.True(t, handlerCalled)
			},
		},
		{
			name: "no token provided",
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusForbidden, r.StatusCode)
				require.False(t, handlerCalled)
			},
		},
		{
			name: "invalid token provided",
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				q := req.URL.Query()
				q.Set("access_token", "bogus-token")
				req.URL.RawQuery = q.Encode()
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusForbidden, r.StatusCode)
				require.False(t, handlerCalled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := testCase.setup()
			handlerCalled := false
			NewTokenFilter(
				testConfig,
				func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				},
			).ServeHTTP(rr, req)
			res := rr.Result()
			defer res.Body.Close()
			testCase.assertions(handlerCalled, res)
		})
	}
}
