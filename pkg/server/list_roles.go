package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
)

// @id ListProjectRoles
// @Summary List project-level Kargo Role virtual resources
// @Description List project-level Kargo Role virtual resources. Returns a
// @Description RoleList resource.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Query as-resources boolean false "Return the roles as their underlying Kubernetes resources"
// @Produce json
// @Success 200 {object} object "RoleList custom resource (rbacapi.RoleList) or underlying resources"
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
			rbacResources, err := s.rolesDB.GetAsResources(
				ctx,
				false,
				project,
				kargoRoleName,
			)
			if err != nil {
				_ = c.Error(err)
				return
			}
			resources[i] = &rbacapi.RoleResources{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: project,
					Name:      kargoRoleName,
				},
				ServiceAccount: *rbacResources.ServiceAccount,
				Roles:          rbacResources.Roles,
				ClusterRoles:   rbacResources.ClusterRoles,
				RoleBindings:   rbacResources.RoleBindings,
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
// @Success 200 {object} object "RoleList custom resource (rbacapi.RoleList) or underlying resources"
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
			rbacResources, err := s.rolesDB.GetAsResources(ctx, true, "", kargoRoleName)
			if err != nil {
				_ = c.Error(err)
				return
			}
			resources[i] = &rbacapi.RoleResources{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: rbacResources.ServiceAccount.Namespace,
					Name:      kargoRoleName,
				},
				ServiceAccount: *rbacResources.ServiceAccount,
				Roles:          rbacResources.Roles,
				ClusterRoles:   rbacResources.ClusterRoles,
				RoleBindings:   rbacResources.RoleBindings,
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
