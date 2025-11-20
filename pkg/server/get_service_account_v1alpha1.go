package server

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetServiceAccount(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetServiceAccountRequest],
) (*connect.Response[svcv1alpha1.GetServiceAccountResponse], error) {
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

	sa, err := s.serviceAccountsDB.Get(ctx, systemLevel, project, name)
	if err != nil {
		return nil, fmt.Errorf("error getting Kubernetes ServiceAccount: %w", err)
	}

	var rawBytes []byte
	switch req.Msg.Format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		if rawBytes, err = json.Marshal(sa); err != nil {
			return nil,
				fmt.Errorf("error marshaling Kargo ServiceAccount to raw JSON: %w", err)
		}
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		if rawBytes, err = sigyaml.Marshal(sa); err != nil {
			return nil,
				fmt.Errorf("error marshaling Kargo ServiceAccount to raw YAML: %w", err)
		}
	default:
		return connect.NewResponse(&svcv1alpha1.GetServiceAccountResponse{
			Result: &svcv1alpha1.GetServiceAccountResponse_ServiceAccount{
				ServiceAccount: sa,
			},
		}), nil
	}

	return connect.NewResponse(&svcv1alpha1.GetServiceAccountResponse{
		Result: &svcv1alpha1.GetServiceAccountResponse_Raw{
			Raw: rawBytes,
		},
	}), nil
}
