package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteClusterAnalysisTemplate(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteClusterAnalysisTemplateRequest],
) (*connect.Response[svcv1alpha1.DeleteClusterAnalysisTemplateResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.client.Delete(ctx, &v1alpha1.ClusterAnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}); err != nil {
		return nil, fmt.Errorf("delete ClusterAnalysisTemplate: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteClusterAnalysisTemplateResponse{}), nil
}
