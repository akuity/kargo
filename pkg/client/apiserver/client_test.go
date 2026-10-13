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
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

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
// last request. Requests carrying one of the rejected Authorization headers
// are refused as the API server refuses a token it cannot verify.
func newServer(
	t *testing.T,
	fixtures serverFixtures,
	last *lastRequest,
	rejected ...string,
) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	// The API server answers in JSON and says so; the generated client
	// decodes nothing it is not told is JSON.
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	// accept records what the request carried and reports whether the handler
	// may go on; a request carrying a rejected credential has been refused.
	accept := func(w http.ResponseWriter, r *http.Request) bool {
		last.auth = r.Header.Get("Authorization")
		last.query = r.URL.RawQuery
		if slices.Contains(rejected, last.auth) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return false
		}
		return true
	}
	// The "broken" Project stands in for a proxy or gateway answering in the
	// API server's place with something other than its JSON error body.
	broken := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>gateway sad</html>"))
	}
	mux.HandleFunc("GET /v1beta1/projects/{project}/targets", func(w http.ResponseWriter, r *http.Request) {
		if !accept(w, r) {
			return
		}
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
		if !accept(w, r) {
			return
		}
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
			if !accept(w, r) {
				return
			}
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
			if !accept(w, r) {
				return
			}
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

// controlPlane stands in for the control plane's Kubernetes API: it answers
// who the caller is (a SelfSubjectReview) and serves the TokenRequest
// subresource, issuing tokens minted-1, minted-2 and so on, and records what
// each request carried. status applies to token requests only.
type controlPlane struct {
	server   *httptest.Server
	status   int
	username string

	mu     sync.Mutex
	auths  []string
	paths  []string
	minted int
}

const testServiceAccount = "system:serviceaccount:kargo:shard-eu"

func newControlPlane(t *testing.T, status int) *controlPlane {
	t.Helper()
	cp := &controlPlane{status: status, username: testServiceAccount}
	record := func(r *http.Request) {
		cp.mu.Lock()
		defer cp.mu.Unlock()
		cp.auths = append(cp.auths, r.Header.Get("Authorization"))
		cp.paths = append(cp.paths, r.URL.Path)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(
		"POST /apis/authentication.k8s.io/v1/selfsubjectreviews",
		func(w http.ResponseWriter, r *http.Request) {
			record(r)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(authnv1.SelfSubjectReview{
				TypeMeta: metav1.TypeMeta{APIVersion: "authentication.k8s.io/v1", Kind: "SelfSubjectReview"},
				Status:   authnv1.SelfSubjectReviewStatus{UserInfo: authnv1.UserInfo{Username: cp.username}},
			})
		},
	)
	mux.HandleFunc(
		"POST /api/v1/namespaces/{namespace}/serviceaccounts/{name}/token",
		func(w http.ResponseWriter, r *http.Request) {
			record(r)
			cp.mu.Lock()
			defer cp.mu.Unlock()
			if cp.status != http.StatusOK {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(cp.status)
				_ = json.NewEncoder(w).Encode(metav1.Status{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
					Status:   metav1.StatusFailure,
					Code:     int32(cp.status),
					Reason:   metav1.StatusReasonForbidden,
					Message:  "no tokens for you",
				})
				return
			}
			cp.minted++
			issued := authnv1.TokenRequest{
				TypeMeta: metav1.TypeMeta{APIVersion: "authentication.k8s.io/v1", Kind: "TokenRequest"},
				Status: authnv1.TokenRequestStatus{
					Token:               "minted-" + strconv.Itoa(cp.minted),
					ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(issued)
		},
	)
	cp.server = httptest.NewServer(mux)
	t.Cleanup(cp.server.Close)
	return cp
}

// config is a REST config for the control plane that authenticates with a
// static token, the simplest of the credentials a kubeconfig can carry.
func (cp *controlPlane) config() *rest.Config {
	return &rest.Config{Host: cp.server.URL, BearerToken: "kubeconfig-token"}
}

func (cp *controlPlane) lastAuth() string {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.auths[len(cp.auths)-1]
}

func (cp *controlPlane) mintedCount() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.minted
}

func (cp *controlPlane) requests() []string {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return slices.Clone(cp.paths)
}

// validConfig is a ClientConfig for the given API server address with the
// token lifetime every client needs.
func validConfig(address string) ClientConfig {
	return ClientConfig{Address: address, TokenLifetime: time.Hour}
}

func TestNew(t *testing.T) {
	t.Parallel()
	cp := newControlPlane(t, http.StatusOK)
	_, err := New(t.Context(), ClientConfig{}, cp.config())
	require.ErrorContains(t, err, "invalid API server address")
	_, err = New(t.Context(), ClientConfig{Address: "kargo-api"}, cp.config())
	require.ErrorContains(t, err, "invalid API server address")
	short := validConfig("http://kargo-api")
	short.TokenLifetime = time.Minute
	_, err = New(t.Context(), short, cp.config())
	require.ErrorContains(t, err, "invalid API server token lifetime")
	missingCA := validConfig("https://kargo-api")
	missingCA.CACertPath = "/nonexistent/ca.crt"
	_, err = New(t.Context(), missingCA, cp.config())
	require.ErrorContains(t, err, "error reading API server CA certificate")
	notPEM := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(notPEM, []byte("not a certificate"), 0o600))
	badCA := validConfig("https://kargo-api")
	badCA.CACertPath = notPEM
	_, err = New(t.Context(), badCA, cp.config())
	require.ErrorContains(t, err, "no certificates found")
	impatient := validConfig("http://kargo-api")
	impatient.Timeout = -time.Second
	_, err = New(t.Context(), impatient, cp.config())
	require.ErrorContains(t, err, "invalid API server timeout")

	// The control plane must answer who the component is before anything
	// else happens.
	_, err = New(t.Context(), validConfig("http://kargo-api"), &rest.Config{Host: "http://127.0.0.1:1"})
	require.ErrorContains(t, err, "error asking the control plane which identity this component has")

	// A credential that is not a ServiceAccount's cannot be acted as.
	user := newControlPlane(t, http.StatusOK)
	user.username = "kubernetes-admin"
	_, err = New(t.Context(), validConfig("http://kargo-api"), user.config())
	require.ErrorContains(t, err, `identifies this component as "kubernetes-admin", which is not a ServiceAccount`)

	c, err := New(t.Context(), validConfig("http://kargo-api/"), cp.config())
	require.NoError(t, err)
	require.Equal(t, "http://kargo-api", c.api.GetConfig().Servers[0].URL)
	// Learning the identity is the only call made so far; no token yet.
	require.Equal(t, []string{"/apis/authentication.k8s.io/v1/selfsubjectreviews"}, cp.requests())
}

func TestClientConfigFromEnv(t *testing.T) {
	t.Setenv("API_SERVER_ADDRESS", "https://kargo-api.kargo.svc")
	t.Setenv("API_SERVER_TIMEOUT", "5s")
	cfg := ClientConfigFromEnv()
	require.Equal(t, "https://kargo-api.kargo.svc", cfg.Address)
	require.Equal(t, 5*time.Second, cfg.Timeout)
	require.Equal(t, time.Hour, cfg.TokenLifetime)
}

// fakeMinter issues the tokens it is given in turn, each valid for lifetime
// from the clock it is handed, or fails with err.
type fakeMinter struct {
	now      func() time.Time
	lifetime time.Duration
	tokens   []string
	err      error
	calls    int
}

func (m *fakeMinter) CreateToken(
	_ context.Context,
	_ string,
	_ *authnv1.TokenRequest,
	_ metav1.CreateOptions,
) (*authnv1.TokenRequest, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	var token string
	if len(m.tokens) > 0 {
		token, m.tokens = m.tokens[0], m.tokens[1:]
	}
	return &authnv1.TokenRequest{Status: authnv1.TokenRequestStatus{
		Token:               token,
		ExpirationTimestamp: metav1.NewTime(m.now().Add(m.lifetime)),
	}}, nil
}

func TestTokenSource(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	minter := &fakeMinter{now: clock, lifetime: time.Hour, tokens: []string{"t1", "t2", "t3"}}
	source := newTokenSource(minter, "shard-eu", time.Hour, time.Second)
	source.now = clock

	token, err := source.Token()
	require.NoError(t, err)
	require.Equal(t, "t1", token.AccessToken)
	require.Equal(t, 1, minter.calls)

	// Well within the lifetime, the token is reused.
	now = now.Add(30 * time.Minute)
	token, err = source.Token()
	require.NoError(t, err)
	require.Equal(t, "t1", token.AccessToken)
	require.Equal(t, 1, minter.calls)

	// Once most of the lifetime has passed, a replacement is minted before
	// the token expires.
	now = now.Add(20 * time.Minute)
	token, err = source.Token()
	require.NoError(t, err)
	require.Equal(t, "t2", token.AccessToken)
	require.Equal(t, 2, minter.calls)

	// A control plane that cannot mint does not take away a token that is
	// still valid.
	minter.err = errors.New("control plane unreachable")
	now = now.Add(55 * time.Minute)
	token, err = source.Token()
	require.NoError(t, err)
	require.Equal(t, "t2", token.AccessToken)
	require.Equal(t, 3, minter.calls)

	// It does once the token has expired.
	now = now.Add(10 * time.Minute)
	_, err = source.Token()
	require.ErrorContains(t, err, `error requesting a token for ServiceAccount "shard-eu"`)
	require.ErrorContains(t, err, "control plane unreachable")

	// Invalidating discards the token: the next request mints anew.
	minter.err = nil
	token, err = source.Token()
	require.NoError(t, err)
	require.Equal(t, "t3", token.AccessToken)
	source.invalidate()
	_, err = source.Token()
	require.ErrorContains(t, err, `issued no token for ServiceAccount "shard-eu"`)
}

func TestNewAPIClient(t *testing.T) {
	t.Parallel()
	var last lastRequest
	server := newServer(t, serverFixtures{targets: map[string][]kargoapi.Target{
		"demo": {{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "us-east"}}},
	}}, &last)
	cp := newControlPlane(t, http.StatusOK)

	api, err := NewAPIClient(t.Context(), validConfig(server.URL), cp.config())
	require.NoError(t, err)

	// A request the Client does not wrap is authenticated the same way: the
	// minted token goes to the API server.
	target, resp, err := api.CoreAPI.GetTarget(t.Context(), "demo", "us-east").Execute()
	require.NoError(t, err)
	_ = resp.Body.Close()
	metadata := target.GetMetadata()
	require.Equal(t, "us-east", metadata.GetName())
	require.Equal(t, "Bearer minted-1", last.auth)

	// The configuration is validated as New validates it.
	_, err = NewAPIClient(t.Context(), ClientConfig{}, cp.config())
	require.ErrorContains(t, err, "invalid API server address")
}

func TestResponseError(t *testing.T) {
	t.Parallel()
	var last lastRequest
	server := newServer(t, serverFixtures{targets: map[string][]kargoapi.Target{"demo": {}}}, &last)
	cp := newControlPlane(t, http.StatusOK)
	api, err := NewAPIClient(t.Context(), validConfig(server.URL), cp.config())
	require.NoError(t, err)

	_, resp, err := api.CoreAPI.GetTarget(t.Context(), "demo", "missing").Execute()
	require.Error(t, err)
	_ = resp.Body.Close()
	var apiErr *APIError
	require.ErrorAs(t, ResponseError(err, resp), &apiErr)
	require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	require.Equal(t, "targets.kargo.akuity.io not found", apiErr.Message)
	require.False(t, IsRetryable(apiErr))

	// An error that never produced a response is returned as is.
	unreachable := errors.New("connection refused")
	require.Same(t, unreachable, ResponseError(unreachable, nil))
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
	fixtures := serverFixtures{
		targets: map[string][]kargoapi.Target{
			"demo":  {euWest, usEast},
			"empty": {},
		},
		rollouts: map[string][]kargoapi.PromotionRollout{
			"demo":  {testRollout, prodRollout},
			"empty": {},
		},
	}
	var last lastRequest
	server := newServer(t, fixtures, &last)
	cp := newControlPlane(t, http.StatusOK)

	t.Run("acts as the ServiceAccount its credential resolves to", func(t *testing.T) {
		own := newControlPlane(t, http.StatusOK)
		c, err := New(t.Context(), validConfig(server.URL), own.config())
		require.NoError(t, err)
		listed, err := c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Len(t, listed, 2)
		require.Equal(t, "us-east", listed[1].Name)
		require.Equal(t, map[string]string{"region": "us"}, listed[1].Labels)
		// The kubeconfig's credential went to the control plane twice: to
		// learn the identity, then to request a token for it. The minted
		// token went to the API server.
		require.Equal(t, []string{
			"/apis/authentication.k8s.io/v1/selfsubjectreviews",
			"/api/v1/namespaces/kargo/serviceaccounts/shard-eu/token",
		}, own.requests())
		require.Equal(t, "Bearer kubeconfig-token", own.lastAuth())
		require.Equal(t, "Bearer minted-1", last.auth)
		// A second request reuses the token: no new identity check, no new
		// token.
		_, err = c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Len(t, own.requests(), 2)
		require.Equal(t, 1, own.mintedCount())
	})

	t.Run("a kubeconfig that authenticates through an exec plugin works too", func(t *testing.T) {
		// The plugin stands in for aws eks get-token and the like: client-go
		// runs it and uses what it prints on every call to the control plane.
		execCP := newControlPlane(t, http.StatusOK)
		c, err := New(t.Context(), validConfig(server.URL), &rest.Config{
			Host: execCP.server.URL,
			ExecProvider: &clientcmdapi.ExecConfig{
				APIVersion:      "client.authentication.k8s.io/v1",
				Command:         "sh",
				Args:            []string{"-c", execCredential("exec-token")},
				InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
			},
		})
		require.NoError(t, err)
		_, err = c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Equal(t, "Bearer exec-token", execCP.lastAuth())
		require.Equal(t, "Bearer minted-1", last.auth)
	})

	t.Run("a token the API server refuses is replaced", func(t *testing.T) {
		var strictLast lastRequest
		strict := newServer(t, fixtures, &strictLast, "Bearer minted-1")
		strictCP := newControlPlane(t, http.StatusOK)
		c, err := New(t.Context(), validConfig(strict.URL), strictCP.config())
		require.NoError(t, err)
		_, err = c.ListTargets(t.Context(), "demo")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
		// The refusal discarded minted-1; the next request carries a new token.
		_, err = c.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
		require.Equal(t, "Bearer minted-2", strictLast.auth)
	})

	t.Run("a control plane that will not mint a token", func(t *testing.T) {
		forbidding := newControlPlane(t, http.StatusForbidden)
		c, err := New(t.Context(), validConfig(server.URL), forbidding.config())
		require.NoError(t, err)
		_, err = c.ListTargets(t.Context(), "demo")
		require.ErrorContains(t, err, `error requesting a token for ServiceAccount "shard-eu"`)
		require.ErrorContains(t, err, "no tokens for you")
		require.False(t, IsRetryable(err))
	})

	t.Run("trusts the CA the config names", func(t *testing.T) {
		tlsServer := httptest.NewTLSServer(server.Config.Handler)
		t.Cleanup(tlsServer.Close)
		caFile := filepath.Join(t.TempDir(), "ca.crt")
		require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: tlsServer.Certificate().Raw,
		}), 0o600))
		untrusting, err := New(t.Context(), validConfig(tlsServer.URL), cp.config())
		require.NoError(t, err)
		_, err = untrusting.ListTargets(t.Context(), "demo")
		require.Error(t, err)
		trustingCfg := validConfig(tlsServer.URL)
		trustingCfg.CACertPath = caFile
		trusting, err := New(t.Context(), trustingCfg, cp.config())
		require.NoError(t, err)
		_, err = trusting.ListTargets(t.Context(), "demo")
		require.NoError(t, err)
	})

	c, err := New(t.Context(), validConfig(server.URL), cp.config())
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
		require.Regexp(t, `^Bearer minted-\d+$`, last.auth)
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
		require.Regexp(t, `^Bearer minted-\d+$`, last.auth)
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
		down, err := New(t.Context(), validConfig("http://127.0.0.1:1"), cp.config())
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
		impatientCfg := validConfig(server.URL)
		impatientCfg.Timeout = 50 * time.Millisecond
		impatient, err := New(t.Context(), impatientCfg, cp.config())
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
			name: "the control plane would not mint a token",
			err: &url.Error{Op: "Get", URL: "http://kargo-api", Err: fmt.Errorf(
				"error requesting a token: %w",
				apierrors.NewForbidden(schema.GroupResource{Resource: "serviceaccounts"}, "shard-eu", nil),
			)},
			retryable: false,
		},
		{
			name: "the control plane could not mint a token",
			err: fmt.Errorf(
				"error requesting a token: %w",
				apierrors.NewServiceUnavailable("etcd is busy"),
			),
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

// execCredential is a shell command printing the ExecCredential an exec
// plugin answers client-go with, carrying the given token.
func execCredential(token string) string {
	return fmt.Sprintf(
		`printf '{"apiVersion":"client.authentication.k8s.io/v1","kind":"ExecCredential","status":{"token":"%s"}}'`,
		token,
	)
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
