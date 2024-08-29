package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetAnalysisTemplateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisTemplateConfigMapRequest],
) (*connect.Response[svcv1alpha1.GetAnalysisTemplateConfigMapResponse], error) {
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

	// Get the ConfigMap from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": corev1.SchemeGroupVersion.String(),
			"kind":       "ConfigMap",
		},
	}
	if err := s.client.Get(ctx, types.NamespacedName{
		Namespace: project,
		Name:      name,
	}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ConfigMap %q not found in namespace %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	if u.GetLabels()[kargoapi.AnalysisEnvLabelKey] != kargoapi.LabelTrueValue {
		err := fmt.Errorf("ConfigMap %q is not allowed for the AnalysisTemplate", name)
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	switch req.Msg.GetFormat() {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON, svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		_, raw, err := objectOrRaw(&u, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateConfigMapResponse{
			Result: &svcv1alpha1.GetAnalysisTemplateConfigMapResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		var cm corev1.ConfigMap
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &cm); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&cm, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateConfigMapResponse{
			Result: &svcv1alpha1.GetAnalysisTemplateConfigMapResponse_ConfigMap{
				ConfigMap: obj,
			},
		}), nil
	}
}
