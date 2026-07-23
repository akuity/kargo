package server

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// createOrUpdateResourceResponse is the response for creating or updating resources
type createOrUpdateResourceResponse struct {
	Results []createOrUpdateResourceResult `json:"results"`
} // @name CreateOrUpdateResourceResponse

// createOrUpdateResourceResult is the result of creating or updating a resource
type createOrUpdateResourceResult struct {
	CreatedResourceManifest map[string]any `json:"createdResourceManifest,omitempty"`
	UpdatedResourceManifest map[string]any `json:"updatedResourceManifest,omitempty"`
	Error                   string         `json:"error,omitempty"`
} // @name CreateOrUpdateResourceResult

// @id UpdateResource
// @Summary Update resources
// @Description Update (or optionally, upsert) one or more Kargo resources from
// YAML or JSON manifests.
// @Tags Resources
// @Security BearerAuth
// @Accept text/plain
// @Produce json
// @Param upsert query bool false "If true, create a resource if it does not exist"
// @Param manifest body string true "YAML or JSON manifest(s)"
// @Success 200 {object} createOrUpdateResourceResponse "Update results"
// @Router /v1beta1/resources [put]
func (s *server) updateResources(c *gin.Context) {
	ctx := c.Request.Context()

	var upsert bool
	if upsertStr := c.Query("upsert"); upsertStr == "true" {
		upsert = true
	}

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
		_ = c.Error(libhttp.ErrorStr(
			"no resources found in request body",
			http.StatusBadRequest,
		))
		return
	}

	createdProjects := map[string]struct{}{}

	results := make([]createOrUpdateResourceResult, 0, len(resources))
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
		result, err := s.updateResource(ctx, cl, &resource, upsert)
		if err != nil && len(resources) == 1 {
			_ = c.Error(err)
			return
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if err == nil && result.CreatedResourceManifest != nil && resource.GroupVersionKind() == projectGVK {
			createdProjects[resource.GetName()] = struct{}{}
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, createOrUpdateResourceResponse{Results: results})
}

func (s *server) updateResource(
	ctx context.Context,
	cl client.Client,
	obj *unstructured.Unstructured,
	upsert bool,
) (createOrUpdateResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return createOrUpdateResourceResult{
			Error: errSecretManagementDisabled.Error(),
		}, errSecretManagementDisabled
	}

	// Block RBAC privilege escalation via this generic path. Applies to both
	// creating and replacing (e.g. rewriting a Role's rules), so it runs before
	// the create/update decision below.
	if err := s.verifyNoEscalation(ctx, obj); err != nil {
		return createOrUpdateResourceResult{Error: err.Error()}, err
	}

	// Note: It would be tempting to blindly attempt creating the resource and
	// then update it instead if it already exists, but many resource types have
	// defaulting and/or validating webhooks and what we do not want is for some
	// error from a webhook to obscure the fact that the resource already exists.
	// So we'll explicitly check if the resource exists and then decide whether to
	// create or update it.
	existingObj := obj.DeepCopy()
	if err := cl.Get(ctx, client.ObjectKeyFromObject(obj), existingObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return createOrUpdateResourceResult{
				Error: fmt.Errorf("get resource: %w", err).Error(),
			}, err
		}
		existingObj = nil
	}

	if existingObj == nil { // Create the resource
		if !upsert {
			return createOrUpdateResourceResult{
				Error: "resource does not exist",
			}, libhttp.ErrorStr("resource does not exist", http.StatusNotFound)
		}
		// Enforce authorization checks the authorizing client cannot perform
		// implicitly (e.g. the "promote" verb required to create a Promotion).
		if err := s.authorizeResourceCreate(ctx, obj); err != nil {
			return createOrUpdateResourceResult{Error: err.Error()}, err
		}
		// Annotate the object with information about the user who created it,
		// if it is of a type for which that annotation is load-bearing.
		annotateResourceWithCreator(ctx, obj)
		if err := cl.Create(ctx, obj); err != nil {
			return createOrUpdateResourceResult{
				Error: fmt.Errorf("create resource: %w", err).Error(),
			}, err
		}
		// Convert the created object to a map for the response
		createdManifest, err := sigyaml.Marshal(obj)
		if err != nil {
			return createOrUpdateResourceResult{
				Error: fmt.Errorf("marshal created manifest: %w", err).Error(),
			}, err
		}
		var manifestMap map[string]any
		if err = sigyaml.Unmarshal(createdManifest, &manifestMap); err != nil {
			return createOrUpdateResourceResult{
				Error: fmt.Errorf("unmarshal created manifest: %w", err).Error(),
			}, err
		}
		return createOrUpdateResourceResult{CreatedResourceManifest: manifestMap}, nil
	}

	// If we get to here, the resource already exists, so we can update it.

	obj.SetResourceVersion(existingObj.GetResourceVersion())
	if err := cl.Update(ctx, obj); err != nil {
		return createOrUpdateResourceResult{
			Error: fmt.Errorf("update resource: %w", err).Error(),
		}, err
	}
	// Convert the updated object to a map for the response
	updatedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return createOrUpdateResourceResult{
			Error: fmt.Errorf("marshal updated manifest: %w", err).Error(),
		}, err
	}
	var manifestMap map[string]any
	if err = sigyaml.Unmarshal(updatedManifest, &manifestMap); err != nil {
		return createOrUpdateResourceResult{
			Error: fmt.Errorf("unmarshal updated manifest: %w", err).Error(),
		}, err
	}
	return createOrUpdateResourceResult{UpdatedResourceManifest: manifestMap}, nil
}
