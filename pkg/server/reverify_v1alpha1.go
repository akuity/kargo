package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func (s *server) Reverify(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ReverifyRequest],
) (*connect.Response[svcv1alpha1.ReverifyResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}
	stage := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stage); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      stage,
	}
	if err := api.ReverifyStageFreight(ctx, s.client, objKey); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.ReverifyResponse{}), nil
}

// @id Reverify
// @Summary Reverify Freight
// @Description Trigger re-verification of the Freight currently in use by a
// @Description Stage.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/verification [post]
func (s *server) reverify(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stage := c.Param("stage")

	if err := api.ReverifyStageFreight(
		ctx,
		s.client,
		client.ObjectKey{Namespace: project, Name: stage},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
