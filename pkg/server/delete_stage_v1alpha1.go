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

func (s *server) DeleteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	if err := s.client.Delete(
		ctx,
		&kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
		},
	); err != nil {
		return nil, fmt.Errorf("delete stage: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteStageResponse{}), nil
}

// @id DeleteStage
// @Summary Delete a Stage
// @Description Delete a Stage resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/stages/{stage} [delete]
func (s *server) deleteStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("stage")

	if err := s.client.Delete(
		ctx,
		&kargoapi.Stage{
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
