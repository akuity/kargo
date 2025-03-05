package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteAnalysisTemplate(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteAnalysisTemplateRequest],
) (*connect.Response[svcv1alpha1.DeleteAnalysisTemplateResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	if err := s.client.Delete(ctx, &v1alpha1.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project,
			Name:      name,
		},
	}); err != nil {
		return nil, fmt.Errorf("delete AnalysisTemplate: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteAnalysisTemplateResponse{}), nil
}
