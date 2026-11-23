package apiserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

var (
	// projectResources are the forms a missing Project is reported in: by
	// resource when read from the Kubernetes API server, and by Kind when
	// read from a controller-runtime cache, as the API server's
	// RequireProject middleware may.
	projectResources = []schema.GroupResource{
		kargoapi.GroupVersion.WithResource("projects").GroupResource(),
		{Group: kargoapi.GroupVersion.Group, Resource: "Project"},
	}
	// projectConfigsResource is also what a webhook receiver lookup that
	// finds no receiver reports as not found.
	projectConfigsResource = kargoapi.GroupVersion.WithResource("projectconfigs").GroupResource()
)

// isNotFound reports whether err is the API server reporting the named
// resource, in any of the given forms, not found. Its error middleware
// answers that with a 404 whose message is the Kubernetes NotFound error's.
// Any other 404, such as from an API server without the route, or a wrong
// address, is not this answer and must not be mistaken for it.
func isNotFound(err error, name string, grs ...schema.GroupResource) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		return false
	}
	for _, gr := range grs {
		if apiErr.Message == apierrors.NewNotFound(gr, name).Error() {
			return true
		}
	}
	return false
}

// GetProjectConfig returns the Project's ProjectConfig, or nil if the Project
// has none or does not exist.
func (c *Client) GetProjectConfig(
	ctx context.Context,
	project string,
) (*kargoapi.ProjectConfig, error) {
	model, resp, err := c.api.CoreAPI.GetProjectConfig(ctx, project).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		err = ResponseError(err, resp)
		if isNotFound(err, project, projectConfigsResource) ||
			isNotFound(err, project, projectResources...) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting ProjectConfig of Project %q: %w", project, err,
		)
	}
	cfg := &kargoapi.ProjectConfig{}
	if err = convert(model, cfg); err != nil {
		return nil, fmt.Errorf("error decoding ProjectConfig of Project %q: %w", project, err)
	}
	return cfg, nil
}

// UpdateProjectConfigStatus reports the outcome of a reconcile of the
// Project's ProjectConfig and returns the names of the webhook receivers the
// API server did not record. A ProjectConfig that is gone, changed
// concurrently or refused is an APIError that IsRetryable does not retry.
func (c *Client) UpdateProjectConfigStatus(
	ctx context.Context,
	project string,
	req wire.StatusRequest,
) ([]string, error) {
	body := kargogen.UpdateProjectConfigStatusRequest{}
	if err := convert(req, &body); err != nil {
		return nil, fmt.Errorf("error encoding status of ProjectConfig of Project %q: %w", project, err)
	}
	out, resp, err := c.api.CoreAPI.
		UpdateProjectConfigStatus(ctx, project).
		Body(body).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error updating status of ProjectConfig of Project %q: %w", project, ResponseError(err, resp),
		)
	}
	return out.SkippedReceivers, nil
}

// LookupWebhookReceiver returns the webhook receiver that owns the path and
// the Project that configures it, or nil if the API server reports that no
// receiver does.
func (c *Client) LookupWebhookReceiver(
	ctx context.Context,
	path string,
) (*wire.WebhookReceiverLookup, error) {
	model, resp, err := c.api.SystemAPI.LookupWebhookReceiver(ctx).Path(path).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		err = ResponseError(err, resp)
		if isNotFound(err, path, projectConfigsResource) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error looking up webhook receiver at path %q: %w", path, err,
		)
	}
	lookup := &wire.WebhookReceiverLookup{}
	if err = convert(model, lookup); err != nil {
		return nil, fmt.Errorf("error decoding webhook receiver at path %q: %w", path, err)
	}
	return lookup, nil
}

type projectConfigGetter struct {
	client *Client
}

// ProjectConfigGetter returns an api.ProjectConfigGetter that reads
// ProjectConfigs through the API server.
func (c *Client) ProjectConfigGetter() api.ProjectConfigGetter {
	return &projectConfigGetter{client: c}
}

func (g *projectConfigGetter) Get(
	ctx context.Context,
	project string,
) (*kargoapi.ProjectConfig, error) {
	return g.client.GetProjectConfig(ctx, project)
}
