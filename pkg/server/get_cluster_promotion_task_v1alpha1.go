package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) GetClusterPromotionTask(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetClusterPromotionTaskRequest],
) (*connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], error) {
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the ClusterPromotionTask from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ClusterPromotionTask",
		},
	}
	if err := s.client.Get(ctx, client.ObjectKey{Name: name}, u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ClusterPromotionTask %q not found", name)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	task, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.ClusterPromotionTask{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetClusterPromotionTaskResponse{
			Result: &svcv1alpha1.GetClusterPromotionTaskResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetClusterPromotionTaskResponse{
		Result: &svcv1alpha1.GetClusterPromotionTaskResponse_PromotionTask{
			PromotionTask: task,
		},
	}), nil
}

// nolint: lll
// @id GetClusterPromotionTask
// @Summary Retrieve a ClusterPromotionTask
// @Description Retrieve a ClusterPromotionTask by name.
// @Tags Core, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Param cluster-promotion-task path string true "ClusterPromotionTask name"
// @Produce json
// @Success 200 {object} object "ClusterPromotionTask custom resource (github.com/akuity/kargo/api/v1alpha1.ClusterPromotionTask)"
// @Router /v1beta1/shared/cluster-promotion-tasks/{cluster-promotion-task} [get]
func (s *server) getClusterPromotionTask(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("cluster-promotion-task")

	task := &kargoapi.ClusterPromotionTask{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name},
		task,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, task)
}
