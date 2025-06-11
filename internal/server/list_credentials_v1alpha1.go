package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) ListCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListCredentialsRequest],
) (*connect.Response[svcv1alpha1.ListCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	credsLabelSelector := labels.NewSelector()

	credsLabelSelectorRequirement, err := labels.NewRequirement(
		kargoapi.LabelKeyCredentialType,
		selection.In,
		[]string{
			kargoapi.LabelValueCredentialTypeGit,
			kargoapi.LabelValueCredentialTypeHelm,
			kargoapi.LabelValueCredentialTypeImage,
		})

	if err != nil {
		return nil, err
	}

	credsLabelSelector = credsLabelSelector.Add(*credsLabelSelectorRequirement)

	var secretsList corev1.SecretList
	if err := s.client.List(
		ctx,
		&secretsList,
		client.InNamespace(req.Msg.GetProject()),
		&client.ListOptions{
			LabelSelector: credsLabelSelector,
		},
	); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(secretsList.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	secrets := make([]*corev1.Secret, len(secretsList.Items))
	for i, secret := range secretsList.Items {
		secrets[i] = sanitizeCredentialSecret(secret)
	}

	return connect.NewResponse(&svcv1alpha1.ListCredentialsResponse{
		Credentials: secrets,
	}), nil
}
