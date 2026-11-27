package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @id DeleteFreight
// @Summary Delete a Freight resource
// @Description Delete a Freight resource from a project's namespace by name or
// @Description alias.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias} [delete]
func (s *server) deleteFreight(c *gin.Context) {
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	// Using storageClient directly because we don't expose DeleteFreight as a freightCLient method
	err := s.storageClient.DeleteFreight(c.Request.Context(), project, nameOrAlias)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
