package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/io"
)

// updateResourceResponse is the response for updating resources
type updateResourceResponse struct {
	Results []updateResourceResult `json:"results"`
} // @name UpdateResourceResponse

// updateResourceResult is the result of updating a resource
type updateResourceResult struct {
	UpdatedResourceManifest map[string]any `json:"updatedResourceManifest,omitempty"`
	Error                   string         `json:"error,omitempty"`
} // @name UpdateResourceResult

func (s *server) UpdateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateResourceRequest],
) (*connect.Response[svcv1alpha1.UpdateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(projects, otherResources...)
	results := make([]*svcv1alpha1.UpdateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		result, err := s.updateResourceInternal(ctx, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		// Convert to protobuf result
		var protoResult *svcv1alpha1.UpdateResourceResult
		if result.Error != "" {
			protoResult = &svcv1alpha1.UpdateResourceResult{
				Result: &svcv1alpha1.UpdateResourceResult_Error{
					Error: result.Error,
				},
			}
		} else {
			manifestBytes, marshalErr := sigyaml.Marshal(result.UpdatedResourceManifest)
			if marshalErr != nil {
				protoResult = &svcv1alpha1.UpdateResourceResult{
					Result: &svcv1alpha1.UpdateResourceResult_Error{
						Error: fmt.Errorf("marshal updated manifest: %w", marshalErr).Error(),
					},
				}
			} else {
				protoResult = &svcv1alpha1.UpdateResourceResult{
					Result: &svcv1alpha1.UpdateResourceResult_UpdatedResourceManifest{
						UpdatedResourceManifest: manifestBytes,
					},
				}
			}
		}
		results = append(results, protoResult)
	}
	return &connect.Response[svcv1alpha1.UpdateResourceResponse]{
		Msg: &svcv1alpha1.UpdateResourceResponse{
			Results: results,
		},
	}, nil
}

// @id UpdateResource
// @Summary Update resources
// @Description Update one or more Kargo resources from YAML or JSON manifests.
// @Tags Resources
// @Security BearerAuth
// @Accept text/plain
// @Produce json
// @Param manifest body string true "YAML or JSON manifest(s)"
// @Success 200 {object} updateResourceResponse "Update results"
// @Router /v2/resources [patch]
func (s *server) updateResources(c *gin.Context) {
	ctx := c.Request.Context()

	// Limit request body to 4MB to prevent DoS
	const maxBodyBytes = 4 * 1024 * 1024
	manifest, err := io.LimitRead(c.Request.Body, maxBodyBytes)
	if err != nil {
		if errors.Is(err, &io.BodyTooLargeError{}) {
			_ = c.Error(libhttp.Error(err, http.StatusRequestEntityTooLarge))
			return
		}
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

	results := make([]updateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		result, err := s.updateResourceInternal(ctx, &resource)
		if err != nil && len(resources) == 1 {
			_ = c.Error(err)
			return
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, updateResourceResponse{Results: results})
}

func (s *server) updateResourceInternal(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (updateResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return updateResourceResult{
			Error: errSecretManagementDisabled.Error(),
		}, nil
	}

	// Note: We don't blindly attempt updating the resource because many resources
	// types have defaulting and/or validating webhooks and what we do not want is
	// for some error from a webhook to obscure the fact that the resource does
	// not exist.
	existingObj := obj.DeepCopy()
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), existingObj); err != nil {
		return updateResourceResult{
			Error: fmt.Errorf("get resource: %w", err).Error(),
		}, err
	}

	// If we get to here, the resource already exists, so we can update it.

	obj.SetResourceVersion(existingObj.GetResourceVersion())
	if err := s.client.Update(ctx, obj); err != nil {
		return updateResourceResult{
			Error: fmt.Errorf("update resource: %w", err).Error(),
		}, err
	}

	// Convert the updated object to a map for the response
	updatedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return updateResourceResult{
			Error: fmt.Errorf("marshal updated manifest: %w", err).Error(),
		}, err
	}
	var manifestMap map[string]any
	if err = sigyaml.Unmarshal(updatedManifest, &manifestMap); err != nil {
		return updateResourceResult{
			Error: fmt.Errorf("unmarshal updated manifest: %w", err).Error(),
		}, err
	}
	return updateResourceResult{UpdatedResourceManifest: manifestMap}, nil
}
