package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/config"
)

func TestConfigureEngine(t *testing.T) {
	trusted := []string{"10.0.0.0/8", "127.0.0.1"}
	testCases := []struct {
		name       string
		cfg        config.ServerConfig
		remoteAddr string
		headers    http.Header
		expected   string
	}{
		{
			name:       "forwarding headers are ignored by default",
			remoteAddr: "203.0.113.7:4321",
			headers: http.Header{
				"X-Forwarded-For": {"198.51.100.1"},
				"X-Real-Ip":       {"198.51.100.2"},
			},
			expected: "203.0.113.7",
		},
		{
			name:       "forwarding headers from an untrusted peer are ignored",
			cfg:        config.ServerConfig{TrustedProxies: trusted, ClientIPHeader: "CF-Connecting-IP"},
			remoteAddr: "203.0.113.7:4321",
			headers: http.Header{
				"X-Forwarded-For":  {"198.51.100.1"},
				"Cf-Connecting-Ip": {"198.51.100.2"},
			},
			expected: "203.0.113.7",
		},
		{
			name:       "rightmost forwarded entry that is not a trusted proxy",
			cfg:        config.ServerConfig{TrustedProxies: trusted},
			remoteAddr: "10.0.0.1:4321",
			// The leftmost entry is whatever the client claimed.
			headers:  http.Header{"X-Forwarded-For": {"198.51.100.1, 203.0.113.7, 10.0.0.2"}},
			expected: "203.0.113.7",
		},
		{
			name:       "X-Real-IP is not believed",
			cfg:        config.ServerConfig{TrustedProxies: trusted},
			remoteAddr: "10.0.0.1:4321",
			headers:    http.Header{"X-Real-Ip": {"198.51.100.2"}},
			expected:   "10.0.0.1",
		},
		{
			name:       "client IP header from a trusted peer wins",
			cfg:        config.ServerConfig{TrustedProxies: trusted, ClientIPHeader: "CF-Connecting-IP"},
			remoteAddr: "10.0.0.1:4321",
			headers: http.Header{
				"X-Forwarded-For":  {"203.0.113.7"},
				"Cf-Connecting-Ip": {"198.51.100.2"},
			},
			expected: "198.51.100.2",
		},
		{
			name:       "invalid client IP header falls back to forwarded chain",
			cfg:        config.ServerConfig{TrustedProxies: trusted, ClientIPHeader: "CF-Connecting-IP"},
			remoteAddr: "10.0.0.1:4321",
			headers: http.Header{
				"X-Forwarded-For":  {"203.0.113.7"},
				"Cf-Connecting-Ip": {"garbage"},
			},
			expected: "203.0.113.7",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			engine := gin.New()
			require.NoError(t, ConfigureEngine(engine, testCase.cfg))
			var clientIP string
			engine.GET("/", func(c *gin.Context) { clientIP = c.ClientIP() })

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = testCase.remoteAddr
			req.Header = testCase.headers
			engine.ServeHTTP(httptest.NewRecorder(), req)

			require.Equal(t, testCase.expected, clientIP)
		})
	}

	t.Run("invalid trusted proxy", func(t *testing.T) {
		err := ConfigureEngine(
			gin.New(),
			config.ServerConfig{TrustedProxies: []string{"proxy.example.com"}},
		)
		require.ErrorContains(t, err, "error setting trusted proxies")
	})
}
