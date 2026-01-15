package server

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetAPIToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAPITokenRequest],
) (*connect.Response[svcv1alpha1.GetAPITokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	tokenSecret, err := s.rolesDB.GetAPIToken(
		ctx, systemLevel, project, name,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting API token Secret: %w", err)
	}

	var rawBytes []byte
	switch req.Msg.Format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		if rawBytes, err = json.Marshal(tokenSecret); err != nil {
			return nil,
				fmt.Errorf("error marshaling API token Secret to raw JSON: %w", err)
		}
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		if rawBytes, err = sigyaml.Marshal(tokenSecret); err != nil {
			return nil,
				fmt.Errorf("error marshaling API token Secret to raw YAML: %w", err)
		}
	default:
		return connect.NewResponse(&svcv1alpha1.GetAPITokenResponse{
			Result: &svcv1alpha1.GetAPITokenResponse_TokenSecret{
				TokenSecret: svcv1alpha1.FromK8sSecret(tokenSecret),
			},
		}), nil
	}

	return connect.NewResponse(&svcv1alpha1.GetAPITokenResponse{
		Result: &svcv1alpha1.GetAPITokenResponse_Raw{
			Raw: rawBytes,
		},
	}), nil
}
