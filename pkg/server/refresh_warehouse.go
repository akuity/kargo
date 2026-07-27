package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id RefreshWarehouse
// @Summary Refresh a Warehouse
// @Description Refresh a Warehouse resource in a project's namespace.
// @Description Refreshing enqueues the resource for reconciliation by its
// @Description corresponding controller.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param warehouse path string true "Warehouse name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/warehouses/{warehouse}/refresh [post]
func (s *server) refreshWarehouse(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	warehouseName := c.Param("warehouse")

	obj := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{Name: warehouseName, Namespace: project},
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), obj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
