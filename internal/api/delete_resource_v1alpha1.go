package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteResourceRequest],
) (*connect.Response[svcv1alpha1.DeleteResourceResponse], error) {
	cluster, namespaced, err := s.parseKubernetesManifest(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "parse manifest"))
	}

	var res []*svcv1alpha1.DeleteResourceResult
	for _, obj := range cluster {
		res = append(res, s.deleteResource(ctx, obj))
	}
	for _, obj := range namespaced {
		if err := s.validateProject(ctx, obj.GetNamespace()); err != nil {
			res = append(res, &svcv1alpha1.DeleteResourceResult{
				Result: &svcv1alpha1.DeleteResourceResult_Error{
					Error: err.Error(),
				},
			})
			continue
		}
		res = append(res, s.deleteResource(ctx, obj))
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
	if err := s.client.Delete(ctx, obj); err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: errors.Wrap(err, "delete resource").Error(),
			},
		}
	}

	deletedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.DeleteResourceResult{
			Result: &svcv1alpha1.DeleteResourceResult_Error{
				Error: errors.Wrap(err, "marshal deleted manifest").Error(),
			},
		}
	}
	return &svcv1alpha1.DeleteResourceResult{
		Result: &svcv1alpha1.DeleteResourceResult_DeletedResourceManifest{
			DeletedResourceManifest: deletedManifest,
		},
	}
}
