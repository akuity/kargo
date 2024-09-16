package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListAnalysisTemplateSecrets(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAnalysisTemplateSecretsRequest],
) (*connect.Response[svcv1alpha1.ListAnalysisTemplateSecretsResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("Argo Rollouts integration is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(project),
		client.MatchingLabels{
			kargoapi.AnalysisRunTemplateLabelKey: kargoapi.AnalysisRunTemplateLabelValueConfig,
		},
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list Secrets: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	secrets := make([]*corev1.Secret, len(list.Items))
	for idx := range list.Items {
		secrets[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListAnalysisTemplateSecretsResponse{
		Secrets: secrets,
	}), nil
}
