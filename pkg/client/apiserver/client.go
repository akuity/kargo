// Package apiserver is the control plane's client for the Kargo API server.
// Some of what Kargo manages, such as Targets and PromotionRollouts, lives in
// the control plane's database rather than in Kubernetes, and only the API
// server serves it. A component may run far from the control plane, on
// infrastructure where the database is not reachable, so such a component
// never opens the database itself; it asks the API server, which authorizes
// it like any other caller.
//
// The client authenticates with the bearer credential of a Kubernetes REST
// config, so a component needs no credential of its own beyond the identity
// it already runs as.
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
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

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
}

// ClientConfigFromEnv returns a ClientConfig populated from the environment.
func ClientConfigFromEnv() ClientConfig {
	cfg := ClientConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Client reads Targets and reads and creates PromotionRollouts through the
// Kargo API server, using the generated client that the CLI uses too.
type Client struct {
	api *kargogen.APIClient
}

// New returns a Client for the API server the config locates that
// authenticates with the bearer credential of the given Kubernetes REST
// config: the controller's own ServiceAccount token in-cluster, or the token
// its control-plane kubeconfig carries. The API server verifies that token
// with a TokenReview and authorizes each request against the controller's
// RBAC, so no credential of its own is needed. A token file is re-read as it
// rotates.
func New(cfg ClientConfig, restCfg *rest.Config) (*Client, error) {
	parsed, err := url.Parse(cfg.Address)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API server address %q", cfg.Address)
	}
	if cfg.Timeout < 0 {
		return nil, fmt.Errorf("invalid API server timeout %s", cfg.Timeout)
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
	rt, err := transport.NewBearerAuthWithRefreshRoundTripper(
		restCfg.BearerToken,
		restCfg.BearerTokenFile,
		base,
	)
	if err != nil {
		return nil, fmt.Errorf("error configuring API server credentials: %w", err)
	}
	genCfg := kargogen.NewConfiguration()
	genCfg.Servers = kargogen.ServerConfigurations{
		{URL: strings.TrimSuffix(cfg.Address, "/")},
	}
	genCfg.HTTPClient = &http.Client{Transport: rt, Timeout: cfg.Timeout}
	return &Client{api: kargogen.NewAPIClient(genCfg)}, nil
}

// ListTargets returns every Target in the Project, ordered by name.
func (c *Client) ListTargets(ctx context.Context, project string) ([]kargoapi.Target, error) {
	list, resp, err := c.api.CoreAPI.ListTargets(ctx, project).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Targets in Project %q: %w", project, apiError(err, resp),
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
			"error getting Target %q in Project %q: %w", name, project, apiError(err, resp),
		)
	}
	target := &kargoapi.Target{}
	if err = convert(model, target); err != nil {
		return nil, fmt.Errorf("error decoding Target %q in Project %q: %w", name, project, err)
	}
	return target, nil
}

// ListPromotionRollouts returns the Stage's PromotionRollouts in the
// Project, or every PromotionRollout in the Project when stage is empty, in
// creation order.
func (c *Client) ListPromotionRollouts(
	ctx context.Context,
	project string,
	stage string,
) ([]kargoapi.PromotionRollout, error) {
	req := c.api.CoreAPI.ListPromotionRollouts(ctx, project)
	if stage != "" {
		req = req.Stage(stage)
	}
	list, resp, err := req.Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error listing PromotionRollouts in Project %q: %w", project, apiError(err, resp),
		)
	}
	rollouts := &kargoapi.PromotionRolloutList{}
	if err = convert(list, rollouts); err != nil {
		return nil, fmt.Errorf("error decoding PromotionRollouts in Project %q: %w", project, err)
	}
	return rollouts.Items, nil
}

// CreatePromotionRollout asks the API server to promote the Freight to the
// Targets the Stage governs now, and returns the PromotionRollout it
// created. The API server resolves the Targets and authorizes the request
// as a promotion to the Stage; IsForbidden tells a refusal apart from a
// failure.
func (c *Client) CreatePromotionRollout(
	ctx context.Context,
	project string,
	stage string,
	freight string,
) (*kargoapi.PromotionRollout, error) {
	model, resp, err := c.api.CoreAPI.CreatePromotionRollout(ctx, project).
		Body(kargogen.PromotionRollout{
			Spec: &kargogen.PromotionRolloutSpec{Stage: &stage, Freight: &freight},
		}).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error creating PromotionRollout for Stage %q in Project %q: %w",
			stage, project, apiError(err, resp),
		)
	}
	rollout := &kargoapi.PromotionRollout{}
	if err = convert(model, rollout); err != nil {
		return nil, fmt.Errorf(
			"error decoding PromotionRollout for Stage %q in Project %q: %w", stage, project, err,
		)
	}
	return rollout, nil
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

// apiError turns a response the generated client refused into an APIError
// carrying the status and the server's explanation. An error that never
// produced a response, such as a failure to connect, is returned as is.
func apiError(err error, resp *http.Response) error {
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
	var netErr net.Error
	return errors.As(err, &netErr) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

// IsForbidden reports whether err is the API server refusing a request for
// want of permission.
func IsForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
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
