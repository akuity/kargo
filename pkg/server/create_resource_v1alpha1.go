package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// createResourceResponse is the response for creating resources
type createResourceResponse struct {
	Results []createResourceResult `json:"results"`
} // @name CreateResourceResponse

// createResourceResult is the result of creating a resource
type createResourceResult struct {
	CreatedResourceManifest map[string]any `json:"createdResourceManifest,omitempty"`
	Error                   string         `json:"error,omitempty"`
} // @name CreateResourceResult

// resourceErrorResponse is the name errorResponse was first published under,
// and the name the spec has to keep: renaming the schema would rename the
// corresponding type in every generated client. This operation is the only one
// that documents an error body, and it refers to this alias rather than to
// errorResponse, which is what makes swag honor the name below.
// nolint: unused
type resourceErrorResponse = errorResponse // @name ResourceErrorResponse

func (s *server) CreateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateResourceRequest],
) (*connect.Response[svcv1alpha1.CreateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(projects, otherResources...)

	createdProjects := map[string]struct{}{}

	results := make([]*svcv1alpha1.CreateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		var cl client.Client = s.client
		if _, ok := createdProjects[resource.GetNamespace()]; ok {
			// Kubernetes ignores metadata.namespace for a cluster-scoped kind, so a
			// spoofed value could bypass authorization there. Confirm real scope via
			// the RESTMapper before trusting it.
			if namespaced, err := s.client.IsObjectNamespaced(&resource); err == nil && namespaced {
				// Creating the Project made the user its owner/admin, but RBAC for it
				// is wired up asynchronously by the management controller. Use the API
				// server's own permissions here so this isn't blocked by that race.
				cl = s.client.InternalClient()
			}
		}
		result, err := s.createResource(ctx, cl, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if err == nil && resource.GroupVersionKind() == projectGVK {
			createdProjects[resource.GetName()] = struct{}{}
		}
		// Convert to protobuf result
		var protoResult *svcv1alpha1.CreateResourceResult
		if result.Error != "" {
			protoResult = &svcv1alpha1.CreateResourceResult{
				Result: &svcv1alpha1.CreateResourceResult_Error{
					Error: result.Error,
				},
			}
		} else {
			manifestBytes, marshalErr := sigyaml.Marshal(result.CreatedResourceManifest)
			if marshalErr != nil {
				protoResult = &svcv1alpha1.CreateResourceResult{
					Result: &svcv1alpha1.CreateResourceResult_Error{
						Error: fmt.Errorf("marshal created manifest: %w", marshalErr).Error(),
					},
				}
			} else {
				protoResult = &svcv1alpha1.CreateResourceResult{
					Result: &svcv1alpha1.CreateResourceResult_CreatedResourceManifest{
						CreatedResourceManifest: manifestBytes,
					},
				}
			}
		}
		results = append(results, protoResult)
	}
	return &connect.Response[svcv1alpha1.CreateResourceResponse]{
		Msg: &svcv1alpha1.CreateResourceResponse{Results: results},
	}, nil
}

// @id CreateResource
// @Summary Create resources
// @Description Create one or more Kargo resources from YAML or JSON manifests.
// @Tags Resources
// @Security BearerAuth
// @Accept text/plain
// @Produce json
// @Param manifest body string true "YAML or JSON manifest(s)"
// @Success 201 {object} createResourceResponse "Created successfully"
// @Failure 409 {object} resourceErrorResponse "Conflict"
// @Router /v1beta1/resources [post]
func (s *server) createResources(c *gin.Context) {
	ctx := c.Request.Context()

	// Note that there's middleware in place that limits the body size, which is
	// why we're not defensive about that here.
	manifest, err := io.ReadAll(c.Request.Body)
	if err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	projects, otherResources, err := splitYAML(manifest)
	if err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}
	resources := append(projects, otherResources...)

	if len(resources) == 0 {
		_ = c.Error(libhttp.Error(
			errors.New("no resources found in request body"),
			http.StatusBadRequest,
		))
		return
	}

	createdProjects := map[string]struct{}{}

	results := make([]createResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		var cl client.Client = s.client
		if _, ok := createdProjects[resource.GetNamespace()]; ok {
			// Kubernetes ignores metadata.namespace for a cluster-scoped kind, so a
			// spoofed value could bypass authorization there. Confirm real scope via
			// the RESTMapper before trusting it.
			if namespaced, err := s.client.IsObjectNamespaced(&resource); err == nil && namespaced {
				// Creating the Project made the user its owner/admin, but RBAC for it
				// is wired up asynchronously by the management controller. Use the API
				// server's own permissions here so this isn't blocked by that race.
				cl = s.client.InternalClient()
			}
		}
		result, err := s.createResource(ctx, cl, &resource)
		if err != nil && len(resources) == 1 {
			_ = c.Error(err)
			return
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if err == nil && resource.GroupVersionKind() == projectGVK {
			createdProjects[resource.GetName()] = struct{}{}
		}
		results = append(results, result)
	}
	c.JSON(http.StatusCreated, createResourceResponse{Results: results})
}

func (s *server) createResource(
	ctx context.Context,
	cl client.Client,
	obj *unstructured.Unstructured,
) (createResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return createResourceResult{
			Error: errSecretManagementDisabled.Error(),
		}, errSecretManagementDisabled
	}

	// Note: We don't blindly attempt creating the resource because many resource
	// types have defaulting and/or validating webhooks and what we do not want is
	// for some error from a webhook to obscure the fact that the resource already
	// exists.
	existingObj := obj.DeepCopy()
	err := cl.Get(ctx, client.ObjectKeyFromObject(obj), existingObj)
	if err == nil {
		// Whoops! This resource already exists. Make the error look just like we'd
		// gotten it from directly calling cl.Create.
		err := apierrors.NewAlreadyExists(
			schema.GroupResource{
				Group:    existingObj.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(existingObj.GetKind()),
			},
			existingObj.GetName(),
		)
		return createResourceResult{
			Error: fmt.Errorf("create resource: %w", err).Error(),
		}, err
	}
	if !apierrors.IsNotFound(err) {
		return createResourceResult{
			Error: fmt.Errorf("get resource: %w", err).Error(),
		}, err
	}

	// If we get to here, the resource does not already exists, so we can create
	// it.

	// Annotate the object with information about the user who created it, if
	// it is of a type for which that annotation is load-bearing.
	annotateResourceWithCreator(ctx, obj)

	if err = cl.Create(ctx, obj); err != nil {
		return createResourceResult{
			Error: fmt.Errorf("create resource: %w", err).Error(),
		}, err
	}

	if obj.GroupVersionKind() == freightGVK && s.sender != nil {
		var freight kargoapi.Freight
		if convertErr := runtime.DefaultUnstructuredConverter.FromUnstructured(
			obj.Object,
			&freight,
		); convertErr == nil {
			var actor string
			eventMsg := "Freight created"
			if u, ok := user.InfoFromContext(ctx); ok {
				actor = api.FormatEventUserActor(u)
				eventMsg += fmt.Sprintf(" by %q", actor)
			}
			evt := event.NewFreightCreated(eventMsg, actor, &freight)
			if sendErr := s.sender.Send(ctx, evt); sendErr != nil {
				logging.LoggerFromContext(ctx).Error(
					sendErr, "error sending FreightCreated event")
			}
		}
	}

	// Convert the created object to a map for the response
	createdManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return createResourceResult{
			Error: fmt.Errorf("marshal created manifest: %w", err).Error(),
		}, err
	}
	var manifestMap map[string]any
	if err = sigyaml.Unmarshal(createdManifest, &manifestMap); err != nil {
		return createResourceResult{
			Error: fmt.Errorf("unmarshal created manifest: %w", err).Error(),
		}, err
	}
	return createResourceResult{CreatedResourceManifest: manifestMap}, nil
}
