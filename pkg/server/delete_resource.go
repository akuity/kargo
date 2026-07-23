package server

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	libhttp "github.com/akuity/kargo/pkg/http"
)

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
		result, err := s.deleteResource(ctx, &resource)
		if err != nil && len(resources) == 1 {
			_ = c.Error(err)
			return
		}
		results = append(results, result)
	}

	c.JSON(http.StatusOK, deleteResourceResponse{Results: results})
}

func (s *server) deleteResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (deleteResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return deleteResourceResult{
			Error: errSecretManagementDisabled.Error(),
		}, errSecretManagementDisabled
	}
	if err := s.client.Delete(ctx, obj); err != nil {
		return deleteResourceResult{
			Error: fmt.Errorf("delete resource: %w", err).Error(),
		}, err
	}
	return deleteResourceResult{DeletedResourceManifest: obj.Object}, nil
}
