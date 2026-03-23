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

func (s *server) GetProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetProjectRequest],
) (*connect.Response[svcv1alpha1.GetProjectResponse], error) {
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the Project from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Project",
		},
	}
	if err := s.client.Get(ctx, client.ObjectKey{Name: name}, u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Project %q not found", name)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	p, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.Project{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetProjectResponse{
			Result: &svcv1alpha1.GetProjectResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetProjectResponse{
		Result: &svcv1alpha1.GetProjectResponse_Project{Project: p},
	}), nil
}

// @id GetProject
// @Summary Retrieve a Project resource
// @Description Retrieve a Project resource.
// @Tags Core, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Success 200 {object} object "Project custom resource (github.com/akuity/kargo/api/v1alpha1.Project)"
// @Router /v1beta1/projects/{project} [get]
func (s *server) getProject(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("project")

	project := &kargoapi.Project{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name},
		project,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, project)
}
