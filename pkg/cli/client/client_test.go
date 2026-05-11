package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server"
	"github.com/akuity/kargo/pkg/x/version"
)

func TestVersionHeaderTransport(t *testing.T) {
	// Create a test server that captures the request headers
	var capturedHeaders http.Header
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	// Create a client with our version header transport
	transport := &versionHeaderTransport{wrapped: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	// Make a request
	resp, err := client.Get(s.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the version header was set
	require.Equal(
		t,
		version.GetVersion().Version,
		capturedHeaders.Get(server.CLIVersionHeader),
	)
}
