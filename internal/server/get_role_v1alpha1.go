package server

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigyaml "sigs.k8s.io/yaml"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/internal/server/rbac"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetRole(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetRoleRequest],
) (*connect.Response[svcv1alpha1.GetRoleResponse], error) {
	project := req.Msg.Project
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	sa, roles, rbs, err := s.rolesDB.GetAsResources(ctx, project, name)
	if err != nil {
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
