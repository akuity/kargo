package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id DeleteWarehouse
// @Summary Delete a Warehouse
// @Description Delete a Warehouse resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param warehouse path string true "Warehouse name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/warehouses/{warehouse} [delete]
func (s *server) deleteWarehouse(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("warehouse")

	if err := s.client.Delete(
		ctx,
		&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
