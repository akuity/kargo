package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateResourceRequest],
) (*connect.Response[svcv1alpha1.UpdateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "parse manifest"))
	}
	resources := append(projects, otherResources...)
	res := make([]*svcv1alpha1.UpdateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		res = append(res, s.updateResource(ctx, &resource))
	}
	return &connect.Response[svcv1alpha1.UpdateResourceResponse]{
		Msg: &svcv1alpha1.UpdateResourceResponse{
			Results: res,
		},
	}, nil
}

func (s *server) updateResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) *svcv1alpha1.UpdateResourceResult {
	currentObj := obj.DeepCopy()
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), currentObj); err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: errors.Wrap(err, "get resource").Error(),
			},
		}
	}

	obj.SetResourceVersion(currentObj.GetResourceVersion())
	if err := s.client.Update(ctx, obj); err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: errors.Wrap(err, "update resource").Error(),
			},
		}
	}

	updatedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: errors.Wrap(err, "marshal updated manifest").Error(),
			},
		}
	}
	return &svcv1alpha1.UpdateResourceResult{
		Result: &svcv1alpha1.UpdateResourceResult_UpdatedResourceManifest{
			UpdatedResourceManifest: updatedManifest,
		},
	}
}
