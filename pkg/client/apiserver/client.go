// Package apiserver is the control plane's client for the Kargo API server.
// Some of what Kargo manages, such as Targets, lives in the control plane's
// database rather than in Kubernetes, and only the API server serves it. A
// component may run far from the control plane, on infrastructure where the
// database is not reachable, so such a component never opens the database
// itself; it asks the API server, which authorizes it like any other caller.
//
// The client authenticates as the ServiceAccount of the control plane that
// the component's Kubernetes credential for the control plane resolves to.
// It learns which one that is by asking the control plane (a
// SelfSubjectReview), then obtains short-lived tokens for it through the
// TokenRequest API and sends those to the API server, which verifies them
// with a TokenReview. The Kubernetes credential may be a projected
// ServiceAccount token, a token in a kubeconfig, a client certificate or an
// exec plugin: client-go presents whichever it is on the requests that learn
// the identity and mint the tokens, so the component holds no credential of
// its own for the API server and nothing names the ServiceAccount but the
// credential itself.
package apiserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
	authnv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

// ClientConfig locates the API server.
type ClientConfig struct {
	// Address is the URL at which the controller reaches the API server, for
	// instance https://kargo-api.kargo.svc. Empty means the controller has
	// no API server, and so no access to Targets.
	Address string `envconfig:"API_SERVER_ADDRESS"`
	// CACertPath is a PEM file holding the CA that signed the API server's
	// certificate, for one the controller would not otherwise trust.
	CACertPath string `envconfig:"API_SERVER_CA_CERT_PATH"`
	// Timeout bounds each request to the API server, from connecting to
	// reading the whole response, so that an API server that accepts a
	// connection and then hangs cannot hold a reconcile open indefinitely.
	// Zero means no bound beyond the request's own context.
	Timeout time.Duration `envconfig:"API_SERVER_TIMEOUT" default:"30s"`
	// TokenLifetime is how long each token the component requests for its
	// ServiceAccount is valid. A token is replaced once most of its lifetime
	// has passed, so the control plane is asked for a new one a few times per
	// lifetime at most. Kubernetes requires at least ten minutes.
	TokenLifetime time.Duration `envconfig:"API_SERVER_TOKEN_LIFETIME" default:"1h"`
}

// ClientConfigFromEnv returns a ClientConfig populated from the environment.
func ClientConfigFromEnv() ClientConfig {
	cfg := ClientConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Client reads Targets from the Kargo API server through the generated
// client that the CLI uses too.
type Client struct {
	api *kargogen.APIClient
}

// New returns a Client for the API server the config locates, acting as the
// ServiceAccount that restCfg, the component's Kubernetes configuration for
// the control plane, authenticates as. See NewAPIClient for how it
// authenticates.
func New(ctx context.Context, cfg ClientConfig, restCfg *rest.Config) (*Client, error) {
	api, err := NewAPIClient(ctx, cfg, restCfg)
	if err != nil {
		return nil, err
	}
	return &Client{api: api}, nil
}

// NewAPIClient returns the generated API client, configured for the API
// server the config locates and authenticated the way Client is, for a
// component that makes requests Client does not offer. Wrap a request's
// error with ResponseError to read what the API server said.
//
// The client acts as the ServiceAccount that restCfg, the component's
// Kubernetes configuration for the control plane, authenticates as. It asks
// the control plane for that
// identity once, here, and fails if the identity is not a ServiceAccount:
// the API server authorizes callers through Kubernetes RBAC, and a token can
// be minted for a ServiceAccount only. From then on it mints tokens for that
// ServiceAccount through the TokenRequest API as they are needed; client-go
// authenticates those requests however restCfg says, so a client certificate
// or an exec plugin serves as well as a token, and the ServiceAccount must
// be allowed to create its own token subresource. The API server verifies
// each token with a TokenReview and authorizes each request against the
// ServiceAccount's RBAC.
func NewAPIClient(
	ctx context.Context,
	cfg ClientConfig,
	restCfg *rest.Config,
) (*kargogen.APIClient, error) {
	parsed, err := url.Parse(cfg.Address)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API server address %q", cfg.Address)
	}
	if cfg.Timeout < 0 {
		return nil, fmt.Errorf("invalid API server timeout %s", cfg.Timeout)
	}
	if cfg.TokenLifetime < minTokenLifetime {
		return nil, fmt.Errorf(
			"invalid API server token lifetime %s: Kubernetes requires at least %s",
			cfg.TokenLifetime, minTokenLifetime,
		)
	}
	base := cleanhttp.DefaultPooledTransport()
	if cfg.CACertPath != "" {
		pem, readErr := os.ReadFile(cfg.CACertPath)
		if readErr != nil {
			return nil, fmt.Errorf("error reading API server CA certificate: %w", readErr)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf(
				"no certificates found in API server CA certificate file %q", cfg.CACertPath,
			)
		}
		base.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	kube, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("error configuring access to the control plane: %w", err)
	}
	namespace, name, err := identity(ctx, kube, cfg.Timeout)
	if err != nil {
		return nil, err
	}
	source := newTokenSource(
		kube.CoreV1().ServiceAccounts(namespace),
		name,
		cfg.TokenLifetime,
		cfg.Timeout,
	)
	rt := &tokenTransport{
		base:   &oauth2.Transport{Source: source, Base: base},
		source: source,
	}
	genCfg := kargogen.NewConfiguration()
	genCfg.Servers = kargogen.ServerConfigurations{
		{URL: strings.TrimSuffix(cfg.Address, "/")},
	}
	genCfg.HTTPClient = &http.Client{Transport: rt, Timeout: cfg.Timeout}
	return kargogen.NewAPIClient(genCfg), nil
}

// serviceAccountPrefix begins the username Kubernetes gives a ServiceAccount:
// system:serviceaccount:<namespace>:<name>.
const serviceAccountPrefix = "system:serviceaccount:"

// identity asks the control plane which ServiceAccount the component's
// credential authenticates as, through a SelfSubjectReview, which every
// authenticated caller may create. A credential that resolves to anything
// but a ServiceAccount is refused: the API server authorizes callers through
// Kubernetes RBAC on a ServiceAccount, and tokens can be minted for one only,
// so a shard must be given a credential for a ServiceAccount of the control
// plane, whatever mechanism carries it.
func identity(
	ctx context.Context,
	kube kubernetes.Interface,
	timeout time.Duration,
) (namespace string, name string, err error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	review, err := kube.AuthenticationV1().SelfSubjectReviews().Create(
		ctx, &authnv1.SelfSubjectReview{}, metav1.CreateOptions{},
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"error asking the control plane which identity this component has: %w", err,
		)
	}
	username := review.Status.UserInfo.Username
	namespace, name, ok := strings.Cut(strings.TrimPrefix(username, serviceAccountPrefix), ":")
	if !strings.HasPrefix(username, serviceAccountPrefix) || !ok || namespace == "" || name == "" {
		return "", "", fmt.Errorf(
			"the control plane identifies this component as %q, which is not a "+
				"ServiceAccount; the API server authorizes a component as a "+
				"ServiceAccount of the control plane, so its control plane "+
				"credential must be one for such a ServiceAccount",
			username,
		)
	}
	return namespace, name, nil
}

// minTokenLifetime is the shortest token lifetime the TokenRequest API
// grants.
const minTokenLifetime = 10 * time.Minute

// tokenSource obtains tokens for a ServiceAccount of the control plane
// through the TokenRequest API and hands out the current one until it is
// due for replacement. A token is replaced once most of its lifetime has
// passed, the way the kubelet refreshes a projected token, so a request is
// never sent with one about to expire. If the control plane cannot mint a
// replacement, the current token serves until it actually expires, so a
// transient failure to reach the control plane costs nothing.
type tokenSource struct {
	minter   tokenMinter
	account  string
	lifetime time.Duration
	timeout  time.Duration
	now      func() time.Time

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// tokenMinter is the part of a ServiceAccount client that issues tokens; a
// typed clientset's ServiceAccountInterface satisfies it.
type tokenMinter interface {
	CreateToken(
		ctx context.Context,
		serviceAccount string,
		request *authnv1.TokenRequest,
		opts metav1.CreateOptions,
	) (*authnv1.TokenRequest, error)
}

var _ tokenMinter = (typedcorev1.ServiceAccountInterface)(nil)

var _ oauth2.TokenSource = (*tokenSource)(nil)

func newTokenSource(
	minter tokenMinter,
	account string,
	lifetime time.Duration,
	timeout time.Duration,
) *tokenSource {
	return &tokenSource{
		minter:   minter,
		account:  account,
		lifetime: lifetime,
		timeout:  timeout,
		now:      time.Now,
	}
}

// refreshFraction is the share of a token's lifetime after which the next
// request replaces it.
const refreshFraction = 0.8

// Token implements oauth2.TokenSource.
func (s *tokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	refreshAt := s.expiry.Add(-time.Duration(float64(s.lifetime) * (1 - refreshFraction)))
	if s.token != "" && now.Before(refreshAt) {
		return &oauth2.Token{AccessToken: s.token, Expiry: s.expiry}, nil
	}
	token, expiry, err := s.mint()
	if err != nil {
		if s.token != "" && now.Before(s.expiry) {
			return &oauth2.Token{AccessToken: s.token, Expiry: s.expiry}, nil
		}
		return nil, err
	}
	s.token, s.expiry = token, expiry
	return &oauth2.Token{AccessToken: token, Expiry: expiry}, nil
}

// mint asks the control plane for a new token.
func (s *tokenSource) mint() (string, time.Time, error) {
	ctx := context.Background()
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}
	request := &authnv1.TokenRequest{
		Spec: authnv1.TokenRequestSpec{
			ExpirationSeconds: ptrTo(int64(s.lifetime / time.Second)),
		},
	}
	issued, err := s.minter.CreateToken(ctx, s.account, request, metav1.CreateOptions{})
	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"error requesting a token for ServiceAccount %q: %w", s.account, err,
		)
	}
	if issued.Status.Token == "" {
		return "", time.Time{}, fmt.Errorf(
			"the control plane issued no token for ServiceAccount %q", s.account,
		)
	}
	return issued.Status.Token, issued.Status.ExpirationTimestamp.Time, nil
}

// invalidate drops the current token so that the next request mints a new
// one. The API server refusing a token the control plane issued means the
// ServiceAccount changed under it, which a fresh token resolves.
func (s *tokenSource) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.expiry = "", time.Time{}
}

func ptrTo[T any](v T) *T { return &v }

// tokenTransport sends requests with the token source's current token and
// discards that token when the API server refuses it.
type tokenTransport struct {
	base   http.RoundTripper
	source *tokenSource
}

var _ http.RoundTripper = (*tokenTransport)(nil)

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		t.source.invalidate()
	}
	return resp, err
}

// ListTargets returns every Target in the Project, ordered by name.
func (c *Client) ListTargets(ctx context.Context, project string) ([]kargoapi.Target, error) {
	list, resp, err := c.api.CoreAPI.ListTargets(ctx, project).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Targets in Project %q: %w", project, ResponseError(err, resp),
		)
	}
	targets := &kargoapi.TargetList{}
	if err = convert(list, targets); err != nil {
		return nil, fmt.Errorf("error decoding Targets in Project %q: %w", project, err)
	}
	return targets.Items, nil
}

// GetTarget returns the named Target, or nil if there is no such Target.
func (c *Client) GetTarget(ctx context.Context, project, name string) (*kargoapi.Target, error) {
	model, resp, err := c.api.CoreAPI.GetTarget(ctx, project, name).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Target %q in Project %q: %w", name, project, ResponseError(err, resp),
		)
	}
	target := &kargoapi.Target{}
	if err = convert(model, target); err != nil {
		return nil, fmt.Errorf("error decoding Target %q in Project %q: %w", name, project, err)
	}
	return target, nil
}

// convert re-encodes a model of the generated client as the API type it was
// generated from. The two agree on their JSON, which is the only form either
// is ever exchanged in.
func convert(model, api any) error {
	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, api)
}

// ResponseError turns an error the generated client returned, together with
// the response it came with, into an APIError carrying the status and the
// server's explanation, which IsRetryable then classifies. An error that
// never produced a response, such as a failure to connect, is returned as is.
func ResponseError(err error, resp *http.Response) error {
	genErr := &kargogen.GenericOpenAPIError{}
	if resp == nil || !errors.As(err, &genErr) {
		return err
	}
	return newAPIError(resp.StatusCode, genErr.Body())
}

// APIError is a response from the API server other than success.
type APIError struct {
	StatusCode int
	Message    string
}

// IsRetryable reports whether err is a failure to reach the API server that
// may clear on its own: a timeout or other network error, a canceled
// context, or a response of 5xx or 429, which say the server could not answer
// this time. Any other response is the server's deliberate answer, and a body
// the client could not decode is one it would fail to decode again, so the
// same request would fail the same way; a caller should not keep trying.
func IsRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= http.StatusInternalServerError ||
			apiErr.StatusCode == http.StatusTooManyRequests
	}
	// The control plane answering the token request is a server too, and its
	// deliberate refusals, an identity not allowed to request the token say,
	// are as final as the API server's.
	var status apierrors.APIStatus
	if errors.As(err, &status) {
		code := status.Status().Code
		return code >= http.StatusInternalServerError || code == http.StatusTooManyRequests
	}
	var netErr net.Error
	return errors.As(err, &netErr) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API server responded %d", e.StatusCode)
	}
	return fmt.Sprintf("API server responded %d: %s", e.StatusCode, e.Message)
}

// newAPIError reads the API server's error body, which is {"error": "..."},
// falling back to the status alone when the body is something else.
func newAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode}
	var envelope struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error != "" {
		apiErr.Message = envelope.Error
	} else if text := strings.TrimSpace(string(body)); text != "" {
		apiErr.Message = text
	}
	return apiErr
}
