package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetConfigMapRequest],
) (*connect.Response[svcv1alpha1.GetConfigMapResponse], error) {
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

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the ConfigMap from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: namespace}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ConfigMap %q not found", name)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	cfg, raw, err := objectOrRaw(
		s.client,
		u,
		req.Msg.GetFormat(),
		&corev1.ConfigMap{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetConfigMapResponse{
			Result: &svcv1alpha1.GetConfigMapResponse_Raw{Raw: raw},
		}), nil
	}

	// Convert k8s ConfigMap to proto ConfigMap
	protoConfigMap := svcv1alpha1.FromK8sConfigMap(cfg)

	return connect.NewResponse(&svcv1alpha1.GetConfigMapResponse{
		Result: &svcv1alpha1.GetConfigMapResponse_ConfigMap{ConfigMap: protoConfigMap},
	}), nil
}
