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

func (s *server) AbortVerification(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AbortVerificationRequest],
) (*connect.Response[svcv1alpha1.AbortVerificationResponse], error) {
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
	if err := api.AbortStageFreightVerification(ctx, s.client, objKey); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.AbortVerificationResponse{}), nil
}

// @id AbortVerification
// @Summary Abort a running Verification process
// @Description Abort a running Verification process.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/verification/abort [post]
func (s *server) abortVerification(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stage := c.Param("stage")

	if err := api.AbortStageFreightVerification(
		ctx,
		s.client,
		client.ObjectKey{Namespace: project, Name: stage},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
