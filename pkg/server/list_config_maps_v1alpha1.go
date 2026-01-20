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

func (s *server) ListConfigMaps(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListConfigMapsRequest],
) (*connect.Response[svcv1alpha1.ListConfigMapsResponse], error) {
	var namespace string
	if req.Msg.SystemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		project := req.Msg.Project
		if project != "" {
			if err := s.validateProjectExists(ctx, project); err != nil {
				return nil, err
			}
		}
		namespace = project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}

	var configMapsList corev1.ConfigMapList
	if err := s.client.List(
		ctx,
		&configMapsList,
		client.InNamespace(namespace),
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

	return connect.NewResponse(&svcv1alpha1.ListConfigMapsResponse{
		ConfigMaps: cmPtrs,
	}), nil
}
