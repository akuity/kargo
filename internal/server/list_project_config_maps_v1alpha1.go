package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListProjectConfigMaps(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectConfigMapsRequest],
) (*connect.Response[svcv1alpha1.ListProjectConfigMapsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var configMapsList corev1.ConfigMapList
	if err := s.client.List(
		ctx,
		&configMapsList,
		client.InNamespace(req.Msg.GetProject()),
	); err != nil {
		return nil, fmt.Errorf("list configmaps: %w", err)
	}

	configMaps := configMapsList.Items
	slices.SortFunc(configMaps, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	cmPtrs := []*corev1.ConfigMap{}
	for _, cm := range configMaps {
		cmPtrs = append(cmPtrs, cm.DeepCopy())
	}

	return connect.NewResponse(&svcv1alpha1.ListProjectConfigMapsResponse{
		ConfigMaps: cmPtrs,
	}), nil
}
