package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetConfigMapRequest],
) (*connect.Response[svcv1alpha1.GetConfigMapResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the ConfigMap from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
		},
	}
	if err := s.client.Get(ctx, client.ObjectKey{Name: name, Namespace: project}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ConfigMap %q not found", name)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	switch req.Msg.GetFormat() {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON, svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		_, raw, err := objectOrRaw(&u, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetConfigMapResponse{
			Result: &svcv1alpha1.GetConfigMapResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		p := corev1.ConfigMap{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &p); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&p, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetConfigMapResponse{
			Result: &svcv1alpha1.GetConfigMapResponse_ConfigMap{
				ConfigMap: obj,
			},
		}), nil
	}
}
