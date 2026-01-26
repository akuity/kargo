package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigyaml "sigs.k8s.io/yaml"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/rbac"
)

func (s *server) GetRole(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetRoleRequest],
) (*connect.Response[svcv1alpha1.GetRoleResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, systemLevel, project, name)
	if err != nil {
		if systemLevel {
			return nil, fmt.Errorf(
				"error getting Kubernetes resources for system-level Kargo Role %q: %w",
				name, err,
			)
		}
		return nil, fmt.Errorf(
			"error getting Kubernetes resources for Kargo Role %q in project %q: %w",
			name, project, err,
		)
	}

	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	if req.Msg.AsResources {
		resources := &rbacapi.RoleResources{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
			ServiceAccount: *sa,
			Roles:          roles,
			RoleBindings:   rbs,
		}
		return connect.NewResponse(&svcv1alpha1.GetRoleResponse{
			Result: &svcv1alpha1.GetRoleResponse_Resources{
				Resources: resources,
			},
		}), nil
	}

	kargoRole, err := rbac.ResourcesToRole(sa, roles, rbs)
	if err != nil {
		return nil, err
	}

	var rawBytes []byte
	switch req.Msg.Format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		if rawBytes, err = json.Marshal(kargoRole); err != nil {
			return nil, fmt.Errorf("error marshaling Kargo Role to raw JSON: %w", err)
		}
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		if rawBytes, err = sigyaml.Marshal(kargoRole); err != nil {
			return nil, fmt.Errorf("error marshaling Kargo Role to raw YAML: %w", err)
		}
	default:
		return connect.NewResponse(&svcv1alpha1.GetRoleResponse{
			Result: &svcv1alpha1.GetRoleResponse_Role{
				Role: kargoRole,
			},
		}), nil
	}

	return connect.NewResponse(&svcv1alpha1.GetRoleResponse{
		Result: &svcv1alpha1.GetRoleResponse_Raw{
			Raw: rawBytes,
		},
	}), nil
}

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

	sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, false, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// The ServiceAccount is the most critical component of a Kargo Role. If one
	// was not found, the Kargo Role does not exist.
	if sa == nil {
		_ = c.Error(libhttp.ErrorStr("Role not found", http.StatusNotFound))
		return
	}

	if asResources {
		resources := &rbacapi.RoleResources{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
			ServiceAccount: *sa,
			Roles:          roles,
			RoleBindings:   rbs,
		}
		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRole, err := rbac.ResourcesToRole(sa, roles, rbs)
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

	sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, true, "", name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// The ServiceAccount is the most critical component of a Kargo Role. If one
	// was not found, the Kargo Role does not exist.
	if sa == nil {
		_ = c.Error(libhttp.ErrorStr("Role not found", http.StatusNotFound))
		return
	}

	if asResources {
		resources := &rbacapi.RoleResources{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sa.Namespace,
				Name:      name,
			},
			ServiceAccount: *sa,
			Roles:          roles,
			RoleBindings:   rbs,
		}
		c.JSON(http.StatusOK, resources)
		return
	}

	kargoRole, err := rbac.ResourcesToRole(sa, roles, rbs)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, kargoRole)
}
