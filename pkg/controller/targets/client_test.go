package targets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
)

// The client is what the controllers consume it as.
var (
	_ promotion.TargetGetter = (*Client)(nil)
	_ interface {
		ListTargets(context.Context, string) ([]kargoapi.Target, error)
	} = (*Client)(nil)
)

// newServer serves a Project's Targets the way the API server does, recording
// the Authorization header of the last request.
func newServer(t *testing.T, targets map[string][]kargoapi.Target, lastAuth *string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	// The "broken" Project stands in for a proxy or gateway answering in the
	// API server's place with something other than its JSON error body.
	broken := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>gateway sad</html>"))
	}
	mux.HandleFunc("GET /v1beta1/projects/{project}/targets", func(w http.ResponseWriter, r *http.Request) {
		*lastAuth = r.Header.Get("Authorization")
		if r.PathValue("project") == "broken" {
			broken(w)
			return
		}
		items, ok := targets[r.PathValue("project")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "project not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(kargoapi.TargetList{Items: items})
	})
	mux.HandleFunc("GET /v1beta1/projects/{project}/targets/{name}", func(w http.ResponseWriter, r *http.Request) {
		*lastAuth = r.Header.Get("Authorization")
		if r.PathValue("project") == "broken" {
			broken(w)
			return
		}
		for _, target := range targets[r.PathValue("project")] {
			if target.Name == r.PathValue("name") {
				_ = json.NewEncoder(w).Encode(target)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "targets.kargo.akuity.io not found"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	_, err := NewClient("", &rest.Config{})
	require.ErrorContains(t, err, "invalid API server base URL")
	_, err = NewClient("kargo-api", &rest.Config{})
	require.ErrorContains(t, err, "invalid API server base URL")
	_, err = NewClient("http://kargo-api", &rest.Config{BearerTokenFile: "/nonexistent/token"})
	require.ErrorContains(t, err, "error configuring API server credentials")
	c, err := NewClient("http://kargo-api/", &rest.Config{BearerToken: "t"})
	require.NoError(t, err)
	require.Equal(t, "http://kargo-api", c.baseURL)
}

func TestClient(t *testing.T) {
	t.Parallel()
	usEast := kargoapi.Target{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo", Name: "us-east", Labels: map[string]string{"region": "us"},
	}}
	euWest := kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "eu-west"}}
	var lastAuth string
	server := newServer(t, map[string][]kargoapi.Target{
		"demo":  {euWest, usEast},
		"empty": {},
	}, &lastAuth)

	t.Run("authenticates with a static token", func(t *testing.T) {
		c, err := NewClient(server.URL, &rest.Config{BearerToken: "static-token"})
		require.NoError(t, err)
		targets, err := c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Len(t, targets, 2)
		require.Equal(t, "us-east", targets[1].Name)
		require.Equal(t, map[string]string{"region": "us"}, targets[1].Labels)
		require.Equal(t, "Bearer static-token", lastAuth)
	})

	t.Run("authenticates with a token file", func(t *testing.T) {
		tokenFile := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(tokenFile, []byte("file-token\n"), 0o600))
		c, err := NewClient(server.URL, &rest.Config{BearerTokenFile: tokenFile})
		require.NoError(t, err)
		_, err = c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Equal(t, "Bearer file-token", lastAuth)
	})

	c, err := NewClient(server.URL, &rest.Config{BearerToken: "t"})
	require.NoError(t, err)

	t.Run("a Project with no Targets", func(t *testing.T) {
		targets, err := c.ListTargets(t.Context(), "empty")
		require.NoError(t, err)
		require.Empty(t, targets)
	})

	t.Run("list error carries the server's explanation", func(t *testing.T) {
		_, err := c.ListTargets(t.Context(), "missing")
		require.ErrorContains(t, err, `error listing Targets in Project "missing"`)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
		require.Equal(t, "project not found", apiErr.Message)
	})

	t.Run("a non-JSON error body is still reported", func(t *testing.T) {
		_, err := c.ListTargets(t.Context(), "broken")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		require.Contains(t, apiErr.Message, "gateway sad")
	})

	t.Run("get", func(t *testing.T) {
		target, err := c.GetTarget(t.Context(), "demo", "us-east")
		require.NoError(t, err)
		require.NotNil(t, target)
		require.Equal(t, "us-east", target.Name)
	})

	t.Run("a missing Target is nil, not an error", func(t *testing.T) {
		target, err := c.GetTarget(t.Context(), "demo", "absent")
		require.NoError(t, err)
		require.Nil(t, target)
	})

	t.Run("other get failures are errors", func(t *testing.T) {
		_, err := c.GetTarget(t.Context(), "broken", "x")
		require.ErrorContains(t, err, `error getting Target "x" in Project "broken"`)
	})

	t.Run("an unreachable server is an error", func(t *testing.T) {
		down, err := NewClient("http://127.0.0.1:1", &rest.Config{})
		require.NoError(t, err)
		_, err = down.ListTargets(t.Context(), "demo")
		require.Error(t, err)
		var apiErr *APIError
		require.False(t, errors.As(err, &apiErr))
	})
}
