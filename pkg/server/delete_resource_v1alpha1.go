package server

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) DeleteResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteResourceRequest],
) (*connect.Response[svcv1alpha1.DeleteResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(otherResources, projects...)
	res := make([]*svcv1alpha1.DeleteResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		res = append(res, s.deleteResourceProto(ctx, &resource))
	}
	return &connect.Response[svcv1alpha1.DeleteResourceResponse]{
		Msg: &svcv1alpha1.DeleteResourceResponse{
			Results: res,
		},
	}, nil
}

func (s *server) deleteResourceProto(
	ctx context.Context,
	obj *unstructured.Unstructured,
) *svcv1alpha1.DeleteResourceResult {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: errSecretManagementDisabled.Error(),
			},
		}
	}

	if err := s.client.Delete(ctx, obj); err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: fmt.Errorf("delete resource: %w", err).Error(),
			},
		}
	}

	deletedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: fmt.Errorf("marshal deleted manifest: %w", err).Error(),
			},
		}
	}
	return &svcv1alpha1.DeleteResourceResult{
		Result: &svcv1alpha1.DeleteResourceResult_DeletedResourceManifest{
			DeletedResourceManifest: deletedManifest,
		},
	}
}

type deleteResourceResponse struct {
	Results []deleteResourceResult `json:"results,omitempty"`
} // @name DeleteResourceResponse

type deleteResourceResult struct {
	DeletedResourceManifest map[string]any `json:"deletedResourceManifest,omitempty"`
	Error                   string         `json:"error,omitempty"`
} // @name DeleteResourceResult

// @id DeleteResource
// @Summary Delete resources
// @Description Delete one or more Kargo resources using namespaces and names
// @Description obtained from YAML or JSON manifests.
// @Tags Resources
// @Security BearerAuth
// @Accept text/plain
// @Produce json
// @Param manifest body string true "YAML or JSON manifest(s)"
// @Success 200 {object} deleteResourceResponse
// @Router /v1beta1/resources [delete]
func (s *server) deleteResources(c *gin.Context) {
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
	// Delete other resources first, then projects
	resources := append(otherResources, projects...)

	if len(resources) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"no resources found in request body",
			http.StatusBadRequest,
		))
		return
	}

	results := make([]deleteResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		results = append(results, s.deleteResource(ctx, &resource))
	}

	c.JSON(http.StatusOK, deleteResourceResponse{Results: results})
}

func (s *server) deleteResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) deleteResourceResult {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return deleteResourceResult{
			Error: errSecretManagementDisabled.Error(),
		}
	}
	if err := s.client.Delete(ctx, obj); err != nil {
		return deleteResourceResult{
			Error: fmt.Errorf("delete resource: %w", err).Error(),
		}
	}
	return deleteResourceResult{
		DeletedResourceManifest: obj.Object,
	}
}
