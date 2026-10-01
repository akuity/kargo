// Package targets gives controllers access to Targets, which live in the
// control plane's database and are served by the Kargo API server. A
// controller may run far from the control plane, on infrastructure where the
// database is not reachable, so it never opens the database itself; it asks
// the API server, which authorizes it like any other caller.
package targets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Client reads Targets from the Kargo API server. It satisfies the Stage
// reconciler's TargetLister and the promotion package's TargetGetter.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a Client for the API server at baseURL that authenticates
// with the bearer credential of the given Kubernetes REST config: the
// controller's own ServiceAccount token in-cluster, or the token its
// control-plane kubeconfig carries. The API server verifies that token with a
// TokenReview and authorizes each request against the controller's RBAC, so
// no credential of its own is needed. A token file is re-read as it rotates.
func NewClient(baseURL string, restCfg *rest.Config) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API server base URL %q", baseURL)
	}
	rt, err := transport.NewBearerAuthWithRefreshRoundTripper(
		restCfg.BearerToken,
		restCfg.BearerTokenFile,
		cleanhttp.DefaultPooledTransport(),
	)
	if err != nil {
		return nil, fmt.Errorf("error configuring API server credentials: %w", err)
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Transport: rt},
	}, nil
}

// ListTargets returns every Target in the Project, ordered by name.
func (c *Client) ListTargets(ctx context.Context, project string) ([]kargoapi.Target, error) {
	list := &kargoapi.TargetList{}
	if err := c.get(ctx, c.targetsPath(project), list); err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", project, err)
	}
	return list.Items, nil
}

// GetTarget returns the named Target, or nil if there is no such Target.
func (c *Client) GetTarget(ctx context.Context, project, name string) (*kargoapi.Target, error) {
	target := &kargoapi.Target{}
	err := c.get(ctx, c.targetsPath(project)+"/"+url.PathEscape(name), target)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Target %q in Project %q: %w", name, project, err)
	}
	return target, nil
}

func (c *Client) targetsPath(project string) string {
	return c.baseURL + "/v1beta1/projects/" + url.PathEscape(project) + "/targets"
}

// get performs a GET and decodes a 200 response into out. Any other status is
// returned as an APIError carrying the server's explanation.
func (c *Client) get(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return newAPIError(resp)
	}
	if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("error decoding API server response: %w", err)
	}
	return nil
}

// APIError is a response from the API server other than success.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API server responded %d", e.StatusCode)
	}
	return fmt.Sprintf("API server responded %d: %s", e.StatusCode, e.Message)
}

// newAPIError reads the API server's error body, which is {"error": "..."},
// falling back to the status alone when the body is something else.
func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{StatusCode: resp.StatusCode}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return apiErr
	}
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
