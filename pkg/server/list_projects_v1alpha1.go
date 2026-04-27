package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
)

func (s *server) ListProjects(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	var list kargoapi.ProjectList
	if err := s.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("error listing Projects: %w", err)
	}

	if req.Msg.GetMine() {
		list.Items = filterProjectsByAccess(ctx, list.Items)
	}

	slices.SortFunc(list.Items, func(a, b kargoapi.Project) int {
		return strings.Compare(a.Name, b.Name)
	})

	var filtered []kargoapi.Project
	if req.Msg.GetFilter() != "" {
		filter := strings.ToLower(req.Msg.GetFilter())
		for i := 0; i < len(list.Items); i++ {
			if strings.Contains(strings.ToLower(list.Items[i].Name), filter) {
				filtered = append(filtered, list.Items[i])
			}
		}
		list.Items = filtered
	}

	if len(req.Msg.GetUid()) > 0 {
		for i := 0; i < len(list.Items); i++ {
			if slices.Contains(req.Msg.GetUid(), string(list.Items[i].UID)) {
				filtered = append(filtered, list.Items[i])
			}
		}
		list.Items = filtered
	}

	total := len(list.Items)
	pageSize := len(list.Items)

	// only the starred projects
	if len(req.Msg.GetUid()) > 0 {
		total = len(filtered)
		pageSize = len(filtered)
	}

	if req.Msg.GetPageSize() > 0 {
		pageSize = int(req.Msg.GetPageSize())
	}

	start := int(req.Msg.GetPage()) * pageSize
	end := start + pageSize

	if start >= len(list.Items) {
		return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{}), nil
	}

	if end > len(list.Items) {
		end = len(list.Items)
	}

	list.Items = list.Items[start:end]
	projects := make([]*kargoapi.Project, len(list.Items))
	for i := range list.Items {
		projects[i] = &list.Items[i]
	}
	return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
		Projects: projects,
		Total:    int32(total), // nolint: gosec
	}), nil
}

// @id ListProjects
// @Summary List projects
// @Description List all Projects resources. Returns a ProjectList resource.
// @Tags Core, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Success 200 {object} kargoapi.ProjectList "ProjectList custom resource"
// @Router /v1beta1/projects [get]
func (s *server) listProjects(c *gin.Context) {
	ctx := c.Request.Context()

	list := &kargoapi.ProjectList{}
	if err := s.client.List(ctx, list); err != nil {
		_ = c.Error(err)
		return
	}

	if c.Query("mine") == trueStr {
		list.Items = filterProjectsByAccess(ctx, list.Items)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.Project) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

// filterProjectsByAccess filters the given projects to only those where the
// authenticated user has been mapped to a ServiceAccount in the project's
// namespace.
func filterProjectsByAccess(
	ctx context.Context,
	projects []kargoapi.Project,
) []kargoapi.Project {
	userInfo, _ := user.InfoFromContext(ctx)
	filtered := make([]kargoapi.Project, 0, len(projects))
	for _, project := range projects {
		if _, has := userInfo.ServiceAccountsByNamespace[project.Name]; has {
			filtered = append(filtered, project)
		}
	}
	return filtered
}
