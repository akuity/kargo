package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteAPIToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteAPITokenRequest],
) (*connect.Response[svcv1alpha1.DeleteAPITokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	if err := s.rolesDB.DeleteAPIToken(
		ctx, systemLevel, project, name,
	); err != nil {
		return nil, fmt.Errorf("error deleting API token Secret: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteAPITokenResponse{}), nil
}

// @id DeleteProjectAPIToken
// @Summary Delete a project-level API token
// @Description Delete a project-level API token from a project's namespace.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/api-tokens/{apitoken} [delete]
func (s *server) deleteProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("apitoken")

	if err := s.rolesDB.DeleteAPIToken(ctx, false, project, name); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSystemAPIToken
// @Summary Delete a system-level API token
// @Description Delete a system-level API token.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/api-tokens/{apitoken} [delete]
func (s *server) deleteSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("apitoken")

	if err := s.rolesDB.DeleteAPIToken(ctx, true, "", name); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
