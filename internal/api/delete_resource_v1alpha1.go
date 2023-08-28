package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteResourceRequest],
) (*connect.Response[svcv1alpha1.DeleteResourceResponse], error) {
	parsed, err := s.parseKubernetesManifest(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "parse manifest"))
	}

	// Reverse parsed objects to delete namespaced resources first
	slices.Reverse(parsed)

	var res []*svcv1alpha1.DeleteResourceResult
	for _, obj := range parsed {
		if req.Msg.GetNamespace() != "" {
			obj.SetNamespace(req.Msg.GetNamespace())
		}
		if err := s.validateProject(ctx, obj.GetNamespace()); err != nil {
			res = append(res, &svcv1alpha1.DeleteResourceResult{
				Result: &svcv1alpha1.DeleteResourceResult_Error{
					Error: err.Error(),
				},
			})
			continue
		}
		if err := s.client.Delete(ctx, obj); err != nil {
			res = append(res, &svcv1alpha1.DeleteResourceResult{
				Result: &svcv1alpha1.DeleteResourceResult_Error{
					Error: errors.Wrap(err, "delete resource").Error(),
				},
			})
			continue
		}

		deletedManifest, err := sigyaml.Marshal(obj)
		if err != nil {
			res = append(res, &svcv1alpha1.DeleteResourceResult{
				Result: &svcv1alpha1.DeleteResourceResult_Error{
					Error: errors.Wrap(err, "marshal deleted manifest").Error(),
				},
			})
		}
		res = append(res, &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_DeletedResourceManifest{
				DeletedResourceManifest: deletedManifest,
			},
		})
	}
	return &connect.Response[svcv1alpha1.DeleteResourceResponse]{
		Msg: &svcv1alpha1.DeleteResourceResponse{
			Results: res,
		},
	}, nil
}
