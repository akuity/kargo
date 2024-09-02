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

func (s *server) GetAnalysisTemplateSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisTemplateSecretRequest],
) (*connect.Response[svcv1alpha1.GetAnalysisTemplateSecretResponse], error) {
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

	// Get the Secret from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": corev1.SchemeGroupVersion.String(),
			"kind":       "Secret",
		},
	}
	if err := s.client.Get(ctx, types.NamespacedName{
		Namespace: project,
		Name:      name,
	}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("Secret %q not found in namespace %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	if u.GetLabels()[kargoapi.AnalysisEnvLabelKey] != kargoapi.LabelTrueValue {
		// Hide existence of the Secret for the security
		err := fmt.Errorf("Secret %q not found in namespace %q", name, project)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	switch req.Msg.GetFormat() {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON, svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		_, raw, err := objectOrRaw(&u, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateSecretResponse{
			Result: &svcv1alpha1.GetAnalysisTemplateSecretResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		var secret corev1.Secret
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &secret); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&secret, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateSecretResponse{
			Result: &svcv1alpha1.GetAnalysisTemplateSecretResponse_Secret{
				Secret: obj,
			},
		}), nil
	}
}
