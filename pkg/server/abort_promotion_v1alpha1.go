package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func (s *server) AbortPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AbortPromotionRequest],
) (*connect.Response[svcv1alpha1.AbortPromotionResponse], error) {
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

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	if err := api.AbortPromotion(ctx, s.client, objKey, kargoapi.AbortActionTerminate); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.AbortPromotionResponse{}), nil
}

// @id AbortPromotion
// @Summary Abort a Promotion
// @Description Abort a running Promotion.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param promotion path string true "Promotion name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/promotions/{promotion}/abort [post]
func (s *server) abortPromotion(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("promotion")

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	if err := api.AbortPromotion(ctx, s.client, objKey, kargoapi.AbortActionTerminate); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
