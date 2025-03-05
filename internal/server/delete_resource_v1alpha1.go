package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteResourceRequest],
) (*connect.Response[svcv1alpha1.DeleteResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(otherResources, projects...)
	res := make([]*svcv1alpha1.DeleteResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		res = append(res, s.deleteResource(ctx, &resource))
	}
	return &connect.Response[svcv1alpha1.DeleteResourceResponse]{
		Msg: &svcv1alpha1.DeleteResourceResponse{
			Results: res,
		},
	}, nil
}

func (s *server) deleteResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) *svcv1alpha1.DeleteResourceResult {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: errSecretManagementDisabled.Error(),
			},
		}
	}

	if err := s.client.Delete(ctx, obj); err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: fmt.Errorf("delete resource: %w", err).Error(),
			},
		}
	}

	deletedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: fmt.Errorf("marshal deleted manifest: %w", err).Error(),
			},
		}
	}
	return &svcv1alpha1.DeleteResourceResult{
		Result: &svcv1alpha1.DeleteResourceResult_DeletedResourceManifest{
			DeletedResourceManifest: deletedManifest,
		},
	}
}
