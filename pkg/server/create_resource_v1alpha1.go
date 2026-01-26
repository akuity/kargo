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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
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
			// This resource belongs to a Project we created previously in this API
			// call. The user had sufficient permissions to accomplish that and having
			// done so makes them automatically the "owner" of the Project and an
			// admin. Most of those permissions are wrangled into place asynchronously
			// by the management controller, so in order to proceed with synchronously
			// creating resources within the Project at this time, we will use the API
			// server's own permissions to accomplish that. We accomplish that using
			// s.client's internal client for creation of this resource.
			cl = s.client.InternalClient()
		}
		result, err := s.createResource(ctx, cl, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if resource.GroupVersionKind() == projectGVK {
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
			// This resource belongs to a Project we created previously in this API
			// call. The user had sufficient permissions to accomplish that and having
			// done so makes them automatically the "owner" of the Project and an
			// admin. Most of those permissions are wrangled into place asynchronously
			// by the management controller, so in order to proceed with synchronously
			// creating resources within the Project at this time, we will use the API
			// server's own permissions to accomplish that. We accomplish that using
			// s.client's internal client for creation of this resource.
			cl = s.client.InternalClient()
		}
		result, err := s.createResource(ctx, cl, &resource)
		if err != nil && len(resources) == 1 {
			_ = c.Error(err)
			return
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if resource.GroupVersionKind() == projectGVK {
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
		}, nil
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

	// If the object is a Project, annotate it with information about the user who
	// created it.
	annotateProjectWithCreator(ctx, obj)

	if err = cl.Create(ctx, obj); err != nil {
		return createResourceResult{
			Error: fmt.Errorf("create resource: %w", err).Error(),
		}, err
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
