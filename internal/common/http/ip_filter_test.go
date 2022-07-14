package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewIPFilter(t *testing.T) {
	filter, ok := NewIPFilter(IPFilterConfig{}).(*ipFilter)
	require.True(t, ok)
	require.NotNil(t, filter.config)
}

func TestIPFilter(t *testing.T) {
	_, testNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	testCases := []struct {
		name       string
		filter     *ipFilter
		setup      func() *http.Request
		assertions func(handlerCalled bool, r *http.Response)
	}{
		{
			name: "X-FORWARDED-FOR contains no IP; " +
				"r.RemoteAddr contains invalid IP",
			filter: &ipFilter{},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, "/", nil)
				require.NoError(t, err)
				req.RemoteAddr = "foo" // Not a valid IP, obviously
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusInternalServerError, r.StatusCode)
				require.False(t, handlerCalled)
			},
		},
		{
			name:   "X-FORWARDED-FOR contains invalid IP",
			filter: &ipFilter{},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, "/", nil)
				require.NoError(t, err)
				req.Header.Add("X-FORWARDED-FOR", "foo") // Not a valid IP, obviously
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusInternalServerError, r.StatusCode)
				require.False(t, handlerCalled)
			},
		},
		{
			name: "X-FORWARDED-FOR contains contains valid, disallowed IP",
			filter: &ipFilter{
				config: IPFilterConfig{
					AllowedRanges: []net.IPNet{*testNet},
				},
			},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, "/", nil)
				require.NoError(t, err)
				req.Header.Add("X-FORWARDED-FOR", "192.168.2.125")
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusForbidden, r.StatusCode)
				require.False(t, handlerCalled)
			},
		},
		{
			name: "X-FORWARDED-FOR contains contains valid, allowed IP",
			filter: &ipFilter{
				config: IPFilterConfig{
					AllowedRanges: []net.IPNet{*testNet},
				},
			},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, "/", nil)
				require.NoError(t, err)
				req.Header.Add("X-FORWARDED-FOR", "192.168.1.125")
				return req
			},
			assertions: func(handlerCalled bool, r *http.Response) {
				require.Equal(t, http.StatusOK, r.StatusCode)
				require.True(t, handlerCalled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := testCase.setup()
			handlerCalled := false
			testCase.filter.Decorate(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})(rr, req)
			res := rr.Result()
			defer res.Body.Close()
			testCase.assertions(handlerCalled, res)
		})
	}
}
