package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateResourceRequest],
) (*connect.Response[svcv1alpha1.CreateResourceResponse], error) {
	parsed, err := s.parseKubernetesManifest(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "parse manifest"))
	}

	var res []*svcv1alpha1.CreateResourceResult
	for _, obj := range parsed {
		if req.Msg.GetNamespace() != "" {
			obj.SetNamespace(req.Msg.GetNamespace())
		}
		if err := s.validateProject(ctx, obj.GetNamespace()); err != nil {
			res = append(res, &svcv1alpha1.CreateResourceResult{
				Result: &svcv1alpha1.CreateResourceResult_Error{
					Error: err.Error(),
				},
			})
			continue
		}

		if err := s.client.Create(ctx, obj); err != nil {
			res = append(res, &svcv1alpha1.CreateResourceResult{
				Result: &svcv1alpha1.CreateResourceResult_Error{
					Error: errors.Wrap(err, "create resource").Error(),
				},
			})
			continue
		}

		createdManifest, err := sigyaml.Marshal(obj)
		if err != nil {
			res = append(res, &svcv1alpha1.CreateResourceResult{
				Result: &svcv1alpha1.CreateResourceResult_Error{
					Error: errors.Wrap(err, "marshal created manifest").Error(),
				},
			})
		}
		res = append(res, &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_CreatedResourceManifest{
				CreatedResourceManifest: createdManifest,
			},
		})
	}
	return &connect.Response[svcv1alpha1.CreateResourceResponse]{
		Msg: &svcv1alpha1.CreateResourceResponse{
			Results: res,
		},
	}, nil
}
