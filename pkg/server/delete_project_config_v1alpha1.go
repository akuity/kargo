package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) DeleteProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectConfigRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectConfigResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	if err := s.client.Delete(
		ctx,
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project,
				Namespace: project,
			},
		},
	); err != nil {
		return nil, fmt.Errorf("delete project config: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteProjectConfigResponse{}), nil
}

// @id DeleteProjectConfig
// @Summary Delete a ProjectConfig resource
// @Description Delete the single ProjectConfig resource from a project's
// @Description namespace.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/config [delete]
func (s *server) deleteProjectConfig(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	if err := s.client.Delete(
		ctx,
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project,
				Namespace: project,
			},
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
