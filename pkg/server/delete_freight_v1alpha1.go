package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteFreightRequest],
) (*connect.Response[svcv1alpha1.DeleteFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	alias := req.Msg.GetAlias()
	if (name == "" && alias == "") || (name != "" && alias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of name or alias should not be empty"),
		)
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		name,
		alias,
	)
	if err != nil {
		return nil, err
	}
	if freight == nil {
		if name != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", name, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", alias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := s.client.Delete(ctx, freight); err != nil {
		return nil, fmt.Errorf("delete freight: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteFreightResponse{}), nil
}

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

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	if err := s.client.Delete(c.Request.Context(), freight); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
