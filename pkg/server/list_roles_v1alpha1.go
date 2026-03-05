package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListRoles(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListRolesRequest],
) (*connect.Response[svcv1alpha1.ListRolesResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	kargoRoleNames, err := s.rolesDB.ListNames(ctx, systemLevel, project)
	if err != nil {
		if systemLevel {
			return nil, fmt.Errorf("error listing system-level Kargo Roles: %w", err)
		}
		return nil, fmt.Errorf(
			"error listing Kargo Roles in project %q: %w", project, err,
		)
	}

	if req.Msg.AsResources {
		resources := make([]*rbacapi.RoleResources, len(kargoRoleNames))
		for i, kargoRoleName := range kargoRoleNames {
			sa, roles, rbs, err := s.rolesDB.GetAsResources(
				ctx,
				systemLevel,
				project,
				kargoRoleName,
			)
			if err != nil {
				if systemLevel {
					return nil, fmt.Errorf(
						"error getting Kubernetes resources for system-level Kargo Role %q: %w",
						kargoRoleName, err,
					)
				}
				return nil, fmt.Errorf(
					"error getting Kubernetes resources for Kargo Role %q in project %q: %w",
					kargoRoleName, project, err,
				)
			}
			resources[i] = &rbacapi.RoleResources{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: project,
					Name:      kargoRoleName,
				},
				ServiceAccount: *sa,
				Roles:          roles,
				RoleBindings:   rbs,
			}
		}
		return connect.NewResponse(
			&svcv1alpha1.ListRolesResponse{
				Resources: resources,
			},
		), nil
	}

	kargoRoles := make([]*rbacapi.Role, len(kargoRoleNames))
	for i, kargoRoleName := range kargoRoleNames {
		kargoRole, err := s.rolesDB.Get(ctx, systemLevel, project, kargoRoleName)
		if err != nil {
			if systemLevel {
				return nil, fmt.Errorf(
					"error getting system-level Kargo Role %q: %w", kargoRoleName, err,
				)
			}
			return nil, fmt.Errorf(
				"error getting Kargo Role %q in project %q: %w",
				kargoRoleName, project, err,
			)
		}
		kargoRoles[i] = kargoRole
	}
	return connect.NewResponse(
		&svcv1alpha1.ListRolesResponse{
			Roles: kargoRoles,
		},
	), nil
}

// @id ListProjectRoles
// @Summary List project-level Kargo Role virtual resources
// @Description List project-level Kargo Role virtual resources. Returns a
// @Description RoleList resource.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Query as-resources boolean false "Return the roles as their underlying Kubernetes resources"
// @Produce json
// @Success 200 {object} object "RoleList custom resource (github.com/akuity/kargo/api/rbac/v1alpha1.RoleList)"
// @Router /v1beta1/projects/{project}/roles [get]
func (s *server) listProjectRoles(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	asResources := c.Query("as-resources") == trueStr

	kargoRoleNames, err := s.rolesDB.ListNames(ctx, false, project)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if asResources {
		resources := make([]*rbacapi.RoleResources, len(kargoRoleNames))
		for i, kargoRoleName := range kargoRoleNames {
			sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, false, project, kargoRoleName)
			if err != nil {
				_ = c.Error(err)
				return
			}
			resources[i] = &rbacapi.RoleResources{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: project,
					Name:      kargoRoleName,
				},
				ServiceAccount: *sa,
				Roles:          roles,
				RoleBindings:   rbs,
			}
		}

		// Sort ascending by name
		slices.SortFunc(resources, func(lhs, rhs *rbacapi.RoleResources) int {
			return strings.Compare(lhs.Name, rhs.Name)
		})

		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRoles := make([]*rbacapi.Role, len(kargoRoleNames))
	for i, kargoRoleName := range kargoRoleNames {
		kargoRole, err := s.rolesDB.Get(ctx, false, project, kargoRoleName)
		if err != nil {
			_ = c.Error(err)
			return
		}
		kargoRoles[i] = kargoRole
	}

	// Sort ascending by name
	slices.SortFunc(kargoRoles, func(lhs, rhs *rbacapi.Role) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, kargoRoles)
}

// @id ListSystemRoles
// @Summary List system-level Kargo Role virtual resources
// @Description List system-level Kargo Role virtual resources. Returns a
// @Description RoleList resource.
// @Tags Rbac, System-Level
// @Security BearerAuth
// @Query as-resources boolean false "Return the roles as their underlying Kubernetes resources"
// @Produce json
// @Success 200 {object} object "RoleList custom resource (github.com/akuity/kargo/api/rbac/v1alpha1.RoleList)"
// @Router /v1beta1/system/roles [get]
func (s *server) listSystemRoles(c *gin.Context) {
	ctx := c.Request.Context()
	asResources := c.Query("as-resources") == trueStr

	kargoRoleNames, err := s.rolesDB.ListNames(ctx, true, "")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if asResources {
		resources := make([]*rbacapi.RoleResources, len(kargoRoleNames))
		for i, kargoRoleName := range kargoRoleNames {
			sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, true, "", kargoRoleName)
			if err != nil {
				_ = c.Error(err)
				return
			}
			resources[i] = &rbacapi.RoleResources{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: sa.Namespace,
					Name:      kargoRoleName,
				},
				ServiceAccount: *sa,
				Roles:          roles,
				RoleBindings:   rbs,
			}
		}

		// Sort ascending by name
		slices.SortFunc(resources, func(lhs, rhs *rbacapi.RoleResources) int {
			return strings.Compare(lhs.Name, rhs.Name)
		})

		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRoles := make([]*rbacapi.Role, len(kargoRoleNames))
	for i, kargoRoleName := range kargoRoleNames {
		kargoRole, err := s.rolesDB.Get(ctx, true, "", kargoRoleName)
		if err != nil {
			_ = c.Error(err)
			return
		}
		kargoRoles[i] = kargoRole
	}

	// Sort ascending by name
	slices.SortFunc(kargoRoles, func(lhs, rhs *rbacapi.Role) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, kargoRoles)
}
