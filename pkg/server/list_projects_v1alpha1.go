package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

// listProjectsResponse represents the paginated list of Projects.
type listProjectsResponse struct {
	Items []kargoapi.Project `json:"items"`
	Total int                `json:"total"`
} // @name ListProjectsResponse

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
// @Description List all Projects resources. Supports server-side filtering by
// @Description name substring, by Kubernetes label selector, by UID, and by
// @Description namespaces mapped to the authenticated user's ServiceAccounts,
// @Description plus offset-based pagination.
// @Tags Core, Cluster-Scoped Resource
// @Security BearerAuth
// @Param mine query bool false "Only return Projects whose namespaces are mapped to the user's ServiceAccounts."
// @Param filter query string false "Case-insensitive substring filter applied to the Project name."
// @Param labelSelector query string false "Kubernetes label selector applied to Project labels (e.g. 'env=prod')."
// @Param uid query []string false "Return only Projects whose UID matches one of the given values."
// @Param pageSize query int false "Maximum number of Projects to return. Defaults to all matching Projects."
// @Param page query int false "Zero-indexed page number used together with pageSize."
// @Produce json
// @Success 200 {object} listProjectsResponse
// @Router /v1beta1/projects [get]
func (s *server) listProjects(c *gin.Context) {
	ctx := c.Request.Context()

	var selector labels.Selector
	if rawSelector := c.Query("labelSelector"); rawSelector != "" {
		var err error
		if selector, err = labels.Parse(rawSelector); err != nil {
			_ = c.Error(libhttp.Error(
				fmt.Errorf("invalid labelSelector: %w", err),
				http.StatusBadRequest,
			))
			return
		}
	}

	list := &kargoapi.ProjectList{}
	if err := s.client.List(ctx, list); err != nil {
		_ = c.Error(err)
		return
	}

	if c.Query("mine") == trueStr {
		list.Items = filterProjectsByAccess(ctx, list.Items)
	}

	if selector != nil {
		filtered := list.Items[:0]
		for _, project := range list.Items {
			if selector.Matches(labels.Set(project.Labels)) {
				filtered = append(filtered, project)
			}
		}
		list.Items = filtered
	}

	if filter := strings.ToLower(c.Query("filter")); filter != "" {
		filtered := list.Items[:0]
		for _, project := range list.Items {
			if strings.Contains(strings.ToLower(project.Name), filter) {
				filtered = append(filtered, project)
			}
		}
		list.Items = filtered
	}

	if uids := c.QueryArray("uid"); len(uids) > 0 {
		filtered := list.Items[:0]
		for _, project := range list.Items {
			if slices.Contains(uids, string(project.UID)) {
				filtered = append(filtered, project)
			}
		}
		list.Items = filtered
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.Project) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	total := len(list.Items)

	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	page, _ := strconv.Atoi(c.Query("page"))
	if pageSize > 0 {
		start := page * pageSize
		if start >= total {
			c.JSON(http.StatusOK, listProjectsResponse{Items: []kargoapi.Project{}, Total: total})
			return
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		list.Items = list.Items[start:end]
	}

	c.JSON(http.StatusOK, listProjectsResponse{Items: list.Items, Total: total})
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
