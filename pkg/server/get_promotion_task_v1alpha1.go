package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) GetPromotionTask(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPromotionTaskRequest],
) (*connect.Response[svcv1alpha1.GetPromotionTaskResponse], error) {
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

	// Get the PromotionTask from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "PromotionTask",
		},
	}
	if err := s.client.Get(
		ctx, types.NamespacedName{Namespace: project, Name: name}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("PromotionTask %q not found in namespace %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	task, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.PromotionTask{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetPromotionTaskResponse{
			Result: &svcv1alpha1.GetPromotionTaskResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetPromotionTaskResponse{
		Result: &svcv1alpha1.GetPromotionTaskResponse_PromotionTask{
			PromotionTask: task,
		},
	}), nil
}

// @id GetPromotionTask
// @Summary Retrieve a PromotionTask
// @Description Retrieve a PromotionTask resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param promotion-task path string true "PromotionTask name"
// @Success 200 {object} object "PromotionTask custom resource (github.com/akuity/kargo/api/v1alpha1.PromotionTask)"
// @Router /v1beta1/projects/{project}/promotion-tasks/{promotion-task} [get]
func (s *server) getPromotionTask(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("promotion-task")

	task := &kargoapi.PromotionTask{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		task,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, task)
}
