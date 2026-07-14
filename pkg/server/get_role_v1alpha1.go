package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/rbac"
)

// @id GetProjectRole
// @Summary Retrieve a project-level Kargo Role virtual resource
// @Description Retrieve a project-level Kargo Role virtual resource by name.
// @Description Returns a Kargo Role virtual resource or its underlying
// @Description Kubernetes resources.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param role path string true "Role name"
// @Query as-resources boolean false "Return the role as its underlying Kubernetes resources"
// @Produce json
// @Success 200 {object} object "Role resource (k8s.io/api/rbac/v1.Role) or its underlying Kubernetes resources"
// @Router /v1beta1/projects/{project}/roles/{role} [get]
func (s *server) getProjectRole(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("role")
	asResources := c.Query("as-resources") == trueStr

	resources, err := s.rolesDB.GetAsResources(ctx, false, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// The ServiceAccount is the most critical component of a Kargo Role. If one
	// was not found, the Kargo Role does not exist.
	if resources.ServiceAccount == nil {
		_ = c.Error(libhttp.ErrorStr("Role not found", http.StatusNotFound))
		return
	}

	if asResources {
		resources := &rbacapi.RoleResources{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
			ServiceAccount: *resources.ServiceAccount,
			Roles:          resources.Roles,
			ClusterRoles:   resources.ClusterRoles,
			RoleBindings:   resources.RoleBindings,
		}
		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRole, err := rbac.ResourcesToRole(resources)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, kargoRole)
}

// @id GetSystemRole
// @Summary Retrieve a system-level Kargo Role virtual resource
// @Description Retrieve a system-level Kargo Role virtual resource by name.
// @Description Returns a Kargo Role virtual resource or its underlying
// @Description Kubernetes resources.
// @Tags Rbac, System-Level
// @Security BearerAuth
// @Param role path string true "Role name"
// @Query as-resources boolean false "Return the role as its underlying Kubernetes resources"
// @Produce json
// @Success 200 {object} object "Role resource (k8s.io/api/rbac/v1.Role) or its underlying Kubernetes resources"
// @Router /v1beta1/system/roles/{role} [get]
func (s *server) getSystemRole(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("role")
	asResources := c.Query("as-resources") == trueStr

	resources, err := s.rolesDB.GetAsResources(ctx, true, "", name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// The ServiceAccount is the most critical component of a Kargo Role. If one
	// was not found, the Kargo Role does not exist.
	if resources.ServiceAccount == nil {
		_ = c.Error(libhttp.ErrorStr("Role not found", http.StatusNotFound))
		return
	}

	if asResources {
		resources := &rbacapi.RoleResources{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resources.ServiceAccount.Namespace,
				Name:      name,
			},
			ServiceAccount: *resources.ServiceAccount,
			Roles:          resources.Roles,
			ClusterRoles:   resources.ClusterRoles,
			RoleBindings:   resources.RoleBindings,
		}
		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRole, err := rbac.ResourcesToRole(resources)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, kargoRole)
}
