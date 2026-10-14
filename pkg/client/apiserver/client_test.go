package apiserver

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	_ interface {
		ListPromotionRollouts(context.Context, string, string) ([]kargoapi.PromotionRollout, error)
		CreatePromotionRollout(context.Context, string, string, string) (*kargoapi.PromotionRollout, error)
	} = (*Client)(nil)
)

// serverFixtures is what newServer serves, per Project.
type serverFixtures struct {
	targets  map[string][]kargoapi.Target
	rollouts map[string][]kargoapi.PromotionRollout
}

// lastRequest is what newServer records of the request it last answered.
type lastRequest struct {
	auth  string
	query string
}

// newServer serves a Project's Targets and PromotionRollouts the way the API
// server does, recording the Authorization header and query string of the
// last request.
func newServer(t *testing.T, fixtures serverFixtures, last *lastRequest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	record := func(r *http.Request) {
		last.auth = r.Header.Get("Authorization")
		last.query = r.URL.RawQuery
	}
	// The API server answers in JSON and says so; the generated client
	// decodes nothing it is not told is JSON.
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	// The "broken" Project stands in for a proxy or gateway answering in the
	// API server's place with something other than its JSON error body.
	broken := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>gateway sad</html>"))
	}
	mux.HandleFunc("GET /v1beta1/projects/{project}/targets", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		switch r.PathValue("project") {
		case "broken":
			broken(w)
			return
		case "slow":
			// An API server that accepts the request and never answers; the
			// client's timeout is what ends it.
			<-r.Context().Done()
			return
		}
		items, ok := fixtures.targets[r.PathValue("project")]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
			return
		}
		writeJSON(w, http.StatusOK, kargoapi.TargetList{Items: items})
	})
	mux.HandleFunc("GET /v1beta1/projects/{project}/targets/{name}", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		if r.PathValue("project") == "broken" {
			broken(w)
			return
		}
		for _, target := range fixtures.targets[r.PathValue("project")] {
			if target.Name == r.PathValue("name") {
				writeJSON(w, http.StatusOK, target)
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "targets.kargo.akuity.io not found"})
	})
	mux.HandleFunc(
		"GET /v1beta1/projects/{project}/promotion-rollouts",
		func(w http.ResponseWriter, r *http.Request) {
			record(r)
			if r.PathValue("project") == "broken" {
				broken(w)
				return
			}
			rollouts, ok := fixtures.rollouts[r.PathValue("project")]
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
				return
			}
			// Never nil: the API server serves an empty list as [].
			items := []kargoapi.PromotionRollout{}
			for _, rollout := range rollouts {
				if stage := r.URL.Query().Get("stage"); stage == "" || rollout.Spec.Stage == stage {
					items = append(items, rollout)
				}
			}
			writeJSON(w, http.StatusOK, kargoapi.PromotionRolloutList{Items: items})
		},
	)
	mux.HandleFunc(
		"POST /v1beta1/projects/{project}/promotion-rollouts",
		func(w http.ResponseWriter, r *http.Request) {
			record(r)
			switch r.PathValue("project") {
			case "broken":
				broken(w)
				return
			case "forbidden":
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "not allowed"})
				return
			}
			var requested kargoapi.PromotionRollout
			if err := json.NewDecoder(r.Body).Decode(&requested); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			body := requested.Spec
			writeJSON(w, http.StatusCreated, kargoapi.PromotionRollout{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: r.PathValue("project"),
					Name:      body.Stage + ".01X.abc",
				},
				Spec: kargoapi.PromotionRolloutSpec{
					Stage:   body.Stage,
					Freight: body.Freight,
					Targets: []kargoapi.PromotionRolloutTarget{},
				},
				Status: kargoapi.PromotionRolloutStatus{Phase: kargoapi.PromotionRolloutPhasePending},
			})
		},
	)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestNew(t *testing.T) {
	t.Parallel()
	_, err := New(ClientConfig{}, &rest.Config{})
	require.ErrorContains(t, err, "invalid API server address")
	_, err = New(ClientConfig{Address: "kargo-api"}, &rest.Config{})
	require.ErrorContains(t, err, "invalid API server address")
	_, err = New(
		ClientConfig{Address: "http://kargo-api"},
		&rest.Config{BearerTokenFile: "/nonexistent/token"},
	)
	require.ErrorContains(t, err, "error configuring API server credentials")
	_, err = New(
		ClientConfig{Address: "https://kargo-api", CACertPath: "/nonexistent/ca.crt"},
		&rest.Config{BearerToken: "t"},
	)
	require.ErrorContains(t, err, "error reading API server CA certificate")
	notPEM := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(notPEM, []byte("not a certificate"), 0o600))
	_, err = New(
		ClientConfig{Address: "https://kargo-api", CACertPath: notPEM},
		&rest.Config{BearerToken: "t"},
	)
	require.ErrorContains(t, err, "no certificates found")
	_, err = New(
		ClientConfig{Address: "http://kargo-api", Timeout: -time.Second},
		&rest.Config{BearerToken: "t"},
	)
	require.ErrorContains(t, err, "invalid API server timeout")
	c, err := New(ClientConfig{Address: "http://kargo-api/"}, &rest.Config{BearerToken: "t"})
	require.NoError(t, err)
	require.Equal(t, "http://kargo-api", c.api.GetConfig().Servers[0].URL)
}

func TestClientConfigFromEnv(t *testing.T) {
	t.Setenv("API_SERVER_ADDRESS", "https://kargo-api.kargo.svc")
	t.Setenv("API_SERVER_TIMEOUT", "5s")
	cfg := ClientConfigFromEnv()
	require.Equal(t, "https://kargo-api.kargo.svc", cfg.Address)
	require.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestClient(t *testing.T) {
	t.Parallel()
	usEast := kargoapi.Target{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo", Name: "us-east", Labels: map[string]string{"region": "us"},
	}}
	euWest := kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "eu-west"}}
	testRollout := kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "test.01A.abc"},
		Number:     1,
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:   "test",
			Freight: "fr-1",
			Targets: []kargoapi.PromotionRolloutTarget{{Name: "us-east"}},
		},
		Status: kargoapi.PromotionRolloutStatus{Phase: kargoapi.PromotionRolloutPhaseSucceeded},
	}
	prodRollout := kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "prod.01B.def"},
		Number:     1,
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:   "prod",
			Freight: "fr-1",
			Targets: []kargoapi.PromotionRolloutTarget{},
		},
		Status: kargoapi.PromotionRolloutStatus{Phase: kargoapi.PromotionRolloutPhaseRunning},
	}
	var last lastRequest
	server := newServer(t, serverFixtures{
		targets: map[string][]kargoapi.Target{
			"demo":  {euWest, usEast},
			"empty": {},
		},
		rollouts: map[string][]kargoapi.PromotionRollout{
			"demo":  {testRollout, prodRollout},
			"empty": {},
		},
	}, &last)

	t.Run("authenticates with a static token", func(t *testing.T) {
		c, err := New(ClientConfig{Address: server.URL}, &rest.Config{BearerToken: "static-token"})
		require.NoError(t, err)
		targets, err := c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Len(t, targets, 2)
		require.Equal(t, "us-east", targets[1].Name)
		require.Equal(t, map[string]string{"region": "us"}, targets[1].Labels)
		require.Equal(t, "Bearer static-token", last.auth)
	})

	t.Run("authenticates with a token file", func(t *testing.T) {
		tokenFile := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(tokenFile, []byte("file-token\n"), 0o600))
		c, err := New(ClientConfig{Address: server.URL}, &rest.Config{BearerTokenFile: tokenFile})
		require.NoError(t, err)
		_, err = c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Equal(t, "Bearer file-token", last.auth)
	})

	t.Run("trusts the CA the config names", func(t *testing.T) {
		tlsServer := httptest.NewTLSServer(server.Config.Handler)
		t.Cleanup(tlsServer.Close)
		caFile := filepath.Join(t.TempDir(), "ca.crt")
		require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: tlsServer.Certificate().Raw,
		}), 0o600))
		untrusting, err := New(ClientConfig{Address: tlsServer.URL}, &rest.Config{BearerToken: "t"})
		require.NoError(t, err)
		_, err = untrusting.ListTargets(t.Context(), "demo")
		require.Error(t, err)
		trusting, err := New(
			ClientConfig{Address: tlsServer.URL, CACertPath: caFile},
			&rest.Config{BearerToken: "t"},
		)
		require.NoError(t, err)
		_, err = trusting.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
	})

	c, err := New(ClientConfig{Address: server.URL}, &rest.Config{BearerToken: "t"})
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

	t.Run("lists every PromotionRollout in a Project", func(t *testing.T) {
		rollouts, err := c.ListPromotionRollouts(t.Context(), "demo", "")
		require.NoError(t, err)
		require.Len(t, rollouts, 2)
		require.Equal(t, "test.01A.abc", rollouts[0].Name)
		require.Equal(t, int64(1), rollouts[0].Number)
		require.Equal(t, "test", rollouts[0].Spec.Stage)
		require.Equal(t, "fr-1", rollouts[0].Spec.Freight)
		require.Equal(t, []kargoapi.PromotionRolloutTarget{{Name: "us-east"}}, rollouts[0].Spec.Targets)
		require.Equal(t, kargoapi.PromotionRolloutPhaseSucceeded, rollouts[0].Status.Phase)
		require.Equal(t, "prod.01B.def", rollouts[1].Name)
		// No Stage was asked for, so none was sent.
		require.Empty(t, last.query)
		require.Equal(t, "Bearer t", last.auth)
	})

	t.Run("lists a Stage's PromotionRollouts", func(t *testing.T) {
		rollouts, err := c.ListPromotionRollouts(t.Context(), "demo", "prod")
		require.NoError(t, err)
		require.Len(t, rollouts, 1)
		require.Equal(t, "prod.01B.def", rollouts[0].Name)
		require.Equal(t, kargoapi.PromotionRolloutPhaseRunning, rollouts[0].Status.Phase)
		// The Stage was passed on to the API server rather than filtered here.
		require.Equal(t, "stage=prod", last.query)
	})

	t.Run("a Project with no PromotionRollouts", func(t *testing.T) {
		rollouts, err := c.ListPromotionRollouts(t.Context(), "empty", "")
		require.NoError(t, err)
		require.Empty(t, rollouts)
	})

	t.Run("rollout list error carries the server's explanation", func(t *testing.T) {
		_, err := c.ListPromotionRollouts(t.Context(), "missing", "")
		require.ErrorContains(t, err, `error listing PromotionRollouts in Project "missing"`)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
		require.Equal(t, "project not found", apiErr.Message)
	})

	t.Run("rollout list over a broken gateway", func(t *testing.T) {
		_, err := c.ListPromotionRollouts(t.Context(), "broken", "")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		require.Contains(t, apiErr.Message, "gateway sad")
	})

	t.Run("creates a PromotionRollout for a Stage and Freight", func(t *testing.T) {
		rollout, err := c.CreatePromotionRollout(t.Context(), "demo", "prod", "fr-2")
		require.NoError(t, err)
		require.NotNil(t, rollout)
		require.Equal(t, "prod.01X.abc", rollout.Name)
		require.Equal(t, "demo", rollout.Namespace)
		require.Equal(t, "prod", rollout.Spec.Stage)
		require.Equal(t, "fr-2", rollout.Spec.Freight)
		require.Equal(t, kargoapi.PromotionRolloutPhasePending, rollout.Status.Phase)
		require.Equal(t, "Bearer t", last.auth)
	})

	t.Run("a refused create is forbidden", func(t *testing.T) {
		_, err := c.CreatePromotionRollout(t.Context(), "forbidden", "prod", "fr-2")
		require.ErrorContains(t, err, `error creating PromotionRollout for Stage "prod" in Project "forbidden"`)
		require.ErrorContains(t, err, "not allowed")
		require.True(t, IsForbidden(err))
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.StatusCode)
		require.Equal(t, "not allowed", apiErr.Message)
	})

	t.Run("other create failures are not forbidden", func(t *testing.T) {
		_, err := c.CreatePromotionRollout(t.Context(), "broken", "prod", "fr-2")
		require.False(t, IsForbidden(err))
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		require.Contains(t, apiErr.Message, "gateway sad")
	})

	t.Run("an unreachable server is an error", func(t *testing.T) {
		down, err := New(ClientConfig{Address: "http://127.0.0.1:1"}, &rest.Config{})
		require.NoError(t, err)
		_, err = down.ListTargets(t.Context(), "demo")
		require.Error(t, err)
		var apiErr *APIError
		require.False(t, errors.As(err, &apiErr))
		require.True(t, IsRetryable(err))
		_, err = down.CreatePromotionRollout(t.Context(), "demo", "prod", "fr-2")
		require.Error(t, err)
		require.False(t, IsForbidden(err))
	})

	t.Run("a server that never answers is cut off by the timeout", func(t *testing.T) {
		impatient, err := New(
			ClientConfig{Address: server.URL, Timeout: 50 * time.Millisecond},
			&rest.Config{BearerToken: "t"},
		)
		require.NoError(t, err)
		start := time.Now()
		_, err = impatient.ListTargets(t.Context(), "slow")
		require.Error(t, err)
		require.Less(t, time.Since(start), 5*time.Second)
		require.True(t, IsRetryable(err))
	})
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "the server could not answer",
			err:       fmt.Errorf("listing: %w", &APIError{StatusCode: http.StatusServiceUnavailable}),
			retryable: true,
		},
		{
			name:      "the server asked for a slower pace",
			err:       &APIError{StatusCode: http.StatusTooManyRequests},
			retryable: true,
		},
		{
			name:      "the server refused the credential",
			err:       fmt.Errorf("getting: %w", &APIError{StatusCode: http.StatusUnauthorized}),
			retryable: false,
		},
		{
			name:      "the server refused the request",
			err:       &APIError{StatusCode: http.StatusForbidden},
			retryable: false,
		},
		{
			name:      "the server had no such thing",
			err:       &APIError{StatusCode: http.StatusNotFound},
			retryable: false,
		},
		{
			name:      "the connection failed",
			err:       &url.Error{Op: "Get", URL: "http://kargo-api", Err: &net.OpError{Op: "dial"}},
			retryable: true,
		},
		{
			name:      "the request's context expired",
			err:       fmt.Errorf("getting: %w", context.DeadlineExceeded),
			retryable: true,
		},
		{
			name:      "the response could not be decoded",
			err:       &json.SyntaxError{Offset: 1},
			retryable: false,
		},
		{
			name:      "an error of unknown origin",
			err:       errors.New("something went wrong"),
			retryable: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.retryable, IsRetryable(testCase.err))
		})
	}
}

func TestIsForbidden(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil"},
		{name: "not an API error", err: errors.New("connection refused")},
		{name: "another status", err: &APIError{StatusCode: http.StatusNotFound}},
		{name: "forbidden", err: &APIError{StatusCode: http.StatusForbidden}, expected: true},
		{
			name:     "wrapped forbidden",
			err:      errors.Join(errors.New("outer"), &APIError{StatusCode: http.StatusForbidden}),
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, IsForbidden(testCase.err))
		})
	}
}
