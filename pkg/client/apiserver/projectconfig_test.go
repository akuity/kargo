package apiserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

var _ api.ProjectConfigGetter = (*Client)(nil).ProjectConfigGetter()

// seen is what projectConfigServer recorded of its last request.
type seen struct {
	mu     sync.Mutex
	method string
	path   string
	query  string
	body   []byte
}

// projectConfigServer answers every request with a canned status and body,
// recording the request it last saw. A body of type []byte is written as
// plain text, as Gin answers a request for a route it does not have; any
// other body is encoded as JSON.
func projectConfigServer(
	t *testing.T,
	status int,
	body any,
) (*httptest.Server, *seen) {
	t.Helper()
	s := &seen{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.method, s.path, s.query, s.body = r.Method, r.URL.Path, r.URL.RawQuery, data
		s.mu.Unlock()
		if raw, ok := body.([]byte); ok {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(status)
			_, _ = w.Write(raw)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	return server, s
}

func newTestClient(t *testing.T, address string) *Client {
	t.Helper()
	c, err := New(t.Context(), validConfig(address), newControlPlane(t, http.StatusOK).config())
	require.NoError(t, err)
	return c
}

func TestGetProjectConfig(t *testing.T) {
	t.Parallel()
	cfg := kargoapi.ProjectConfig{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo"}}
	server, last := projectConfigServer(t, http.StatusOK, cfg)
	c := newTestClient(t, server.URL)
	got, err := c.GetProjectConfig(t.Context(), "demo")
	require.NoError(t, err)
	require.Equal(t, "demo", got.Name)
	require.Equal(t, "/v1beta1/projects/demo/config", last.path)

	got, err = c.ProjectConfigGetter().Get(t.Context(), "demo")
	require.NoError(t, err)
	require.Equal(t, "demo", got.Name)

	// The API server's answers for a Project without a ProjectConfig and for
	// a Project that does not exist both mean there is no ProjectConfig. A
	// missing Project read from a controller-runtime cache is reported by
	// Kind.
	for _, gr := range []schema.GroupResource{
		{Group: kargoapi.GroupVersion.Group, Resource: "projectconfigs"},
		{Group: kargoapi.GroupVersion.Group, Resource: "projects"},
		{Group: kargoapi.GroupVersion.Group, Resource: "Project"},
	} {
		server, _ = projectConfigServer(
			t,
			http.StatusNotFound,
			map[string]string{"error": apierrors.NewNotFound(gr, "demo").Error()},
		)
		c = newTestClient(t, server.URL)
		got, err = c.GetProjectConfig(t.Context(), "demo")
		require.NoError(t, err, gr.String())
		require.Nil(t, got)
		got, err = c.ProjectConfigGetter().Get(t.Context(), "demo")
		require.NoError(t, err)
		require.Nil(t, got)
	}

	// Any other 404, such as a route the server does not have, or a not
	// found for something else, is an error.
	for _, body := range []any{
		[]byte("404 page not found"),
		map[string]string{"error": "none"},
		map[string]string{"error": apierrors.NewNotFound(
			schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "projectconfigs"},
			"other",
		).Error()},
	} {
		server, _ = projectConfigServer(t, http.StatusNotFound, body)
		c = newTestClient(t, server.URL)
		_, err = c.GetProjectConfig(t.Context(), "demo")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	}

	server, _ = projectConfigServer(t, http.StatusInternalServerError, map[string]string{"error": "boom"})
	c = newTestClient(t, server.URL)
	_, err = c.GetProjectConfig(t.Context(), "demo")
	require.ErrorContains(t, err, "boom")
}

func TestUpdateProjectConfigStatus(t *testing.T) {
	t.Parallel()
	req := wire.StatusRequest{
		ObservedUID:        "uid-1",
		ObservedGeneration: 3,
		WebhookReceivers:   []kargoapi.WebhookReceiverDetails{{Name: "gh", Path: "/p"}},
	}
	server, last := projectConfigServer(
		t, http.StatusOK, wire.StatusResponse{SkippedReceivers: []string{"x"}},
	)
	c := newTestClient(t, server.URL)
	skipped, err := c.UpdateProjectConfigStatus(t.Context(), "demo", req)
	require.NoError(t, err)
	require.Equal(t, []string{"x"}, skipped)
	require.Equal(t, http.MethodPut, last.method)
	require.Equal(t, "/v1beta1/projects/demo/config/status", last.path)
	sent := wire.StatusRequest{}
	require.NoError(t, json.Unmarshal(last.body, &sent))
	require.Equal(t, req, sent)

	for _, status := range []int{
		http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity,
	} {
		server, _ = projectConfigServer(t, status, map[string]string{"error": "refused"})
		c = newTestClient(t, server.URL)
		_, err = c.UpdateProjectConfigStatus(t.Context(), "demo", req)
		require.ErrorContains(t, err, "refused")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, status, apiErr.StatusCode)
		require.False(t, IsRetryable(err))
	}
}

func TestLookupWebhookReceiver(t *testing.T) {
	t.Parallel()
	found := wire.WebhookReceiverLookup{
		Project:  "demo",
		Receiver: kargoapi.WebhookReceiverDetails{Name: "gh", Path: "/p"},
	}
	// Paths that must reach the server exactly as given.
	for _, path := range []string{
		"/webhook/abc", "a/b%2Fc", "x?y=z&w=1", "with space", "100%", "a+b#c",
	} {
		server, last := projectConfigServer(t, http.StatusOK, found)
		c := newTestClient(t, server.URL)
		got, err := c.LookupWebhookReceiver(t.Context(), path)
		require.NoError(t, err)
		require.Equal(t, "demo", got.Project)
		require.Equal(t, "gh", got.Receiver.Name)
		require.Equal(t, "/v1beta1/system/webhook-receivers", last.path)
		query, err := url.ParseQuery(last.query)
		require.NoError(t, err)
		require.Equal(t, path, query.Get("path"), path)
	}

	// The API server's answer for a path no receiver owns.
	server, _ := projectConfigServer(
		t,
		http.StatusNotFound,
		map[string]string{"error": apierrors.NewNotFound(
			schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "projectconfigs"},
			"/p",
		).Error()},
	)
	c := newTestClient(t, server.URL)
	got, err := c.LookupWebhookReceiver(t.Context(), "/p")
	require.NoError(t, err)
	require.Nil(t, got)

	// Any other 404, such as a route the server does not have yet, is an
	// error.
	for _, body := range []any{
		[]byte("404 page not found"),
		map[string]string{"error": "none"},
	} {
		server, _ = projectConfigServer(t, http.StatusNotFound, body)
		c = newTestClient(t, server.URL)
		_, err = c.LookupWebhookReceiver(t.Context(), "/p")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	}

	server, _ = projectConfigServer(t, http.StatusBadRequest, map[string]string{"error": "bad"})
	c = newTestClient(t, server.URL)
	_, err = c.LookupWebhookReceiver(t.Context(), "")
	require.ErrorContains(t, err, "bad")
}
