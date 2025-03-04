package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListRoles(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListRolesRequest],
) (*connect.Response[svcv1alpha1.ListRolesResponse], error) {
	project := req.Msg.Project
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	kargoRoleNames, err := s.rolesDB.ListNames(ctx, project)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Kargo Roles name in project %q: %w", project, err,
		)
	}

	if req.Msg.AsResources {
		resources := make([]*rbacapi.RoleResources, len(kargoRoleNames))
		for i, kargoRoleName := range kargoRoleNames {
			sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, project, kargoRoleName)
			if err != nil {
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
		kargoRole, err := s.rolesDB.Get(ctx, project, kargoRoleName)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting Kargo Role %q in project %q: %w", kargoRoleName, project, err,
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
