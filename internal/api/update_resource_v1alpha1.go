package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateResourceRequest],
) (*connect.Response[svcv1alpha1.UpdateResourceResponse], error) {
	parsed, err := s.parseKubernetesManifest(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "parse manifest"))
	}

	var res []*svcv1alpha1.UpdateResourceResult
	for _, obj := range parsed {
		if req.Msg.GetNamespace() != "" {
			obj.SetNamespace(req.Msg.GetNamespace())
		}
		if err := s.validateProject(ctx, obj.GetNamespace()); err != nil {
			res = append(res, &svcv1alpha1.UpdateResourceResult{
				Result: &svcv1alpha1.UpdateResourceResult_Error{
					Error: err.Error(),
				},
			})
			continue
		}

		currentObj := obj.DeepCopy()
		if err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), currentObj); err != nil {
			res = append(res, &svcv1alpha1.UpdateResourceResult{
				Result: &svcv1alpha1.UpdateResourceResult_Error{
					Error: errors.Wrap(err, "get resource").Error(),
				},
			})
			continue
		}

		obj.SetResourceVersion(currentObj.GetResourceVersion())
		if err := s.client.Update(ctx, obj); err != nil {
			res = append(res, &svcv1alpha1.UpdateResourceResult{
				Result: &svcv1alpha1.UpdateResourceResult_Error{
					Error: errors.Wrap(err, "update resource").Error(),
				},
			})
			continue
		}

		updatedManifest, err := sigyaml.Marshal(obj)
		if err != nil {
			res = append(res, &svcv1alpha1.UpdateResourceResult{
				Result: &svcv1alpha1.UpdateResourceResult_Error{
					Error: errors.Wrap(err, "marshal updated manifest").Error(),
				},
			})
		}
		res = append(res, &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_UpdatedResourceManifest{
				UpdatedResourceManifest: updatedManifest,
			},
		})
	}
	return &connect.Response[svcv1alpha1.UpdateResourceResponse]{
		Msg: &svcv1alpha1.UpdateResourceResponse{
			Results: res,
		},
	}, nil
}
