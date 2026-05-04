package kargomcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/config"
	generatedclient "github.com/akuity/kargo/pkg/client/generated"
)

// newTestServer starts an httptest.Server with the provided path→handler mux,
// wires the generated API client to it, and returns a Server with the override set.
// The default project is "test-project".
func newTestServer(t *testing.T, mux map[string]http.HandlerFunc) *Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := mux[r.URL.Path]; ok {
			h(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := &generatedclient.TransportConfig{
		Host:     strings.TrimPrefix(srv.URL, "http://"),
		BasePath: "/",
		Schemes:  []string{"http"},
	}
	return &Server{
		cfg:               config.CLIConfig{Project: "test-project"},
		apiClientOverride: generatedclient.NewHTTPClientWithConfig(strfmt.Default, cfg),
	}
}

// jsonOK writes a 200 JSON response with the given body.
func jsonOK(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}
}

// jsonCreated writes a 201 JSON response with the given body.
func jsonCreated(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, body)
	}
}

// structuredContent asserts that result.StructuredContent is a json.RawMessage
// and returns it as a string for inspection.
func structuredContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	sc, ok := result.StructuredContent.(json.RawMessage)
	require.True(t, ok, "expected StructuredContent to be json.RawMessage")
	return string(sc)
}
