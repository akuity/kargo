package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateOrUpdateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateOrUpdateResourceRequest],
) (*connect.Response[svcv1alpha1.CreateOrUpdateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(projects, otherResources...)
	res := make([]*svcv1alpha1.CreateOrUpdateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		res = append(res, s.createOrUpdateResource(ctx, &resource))
	}
	return &connect.Response[svcv1alpha1.CreateOrUpdateResourceResponse]{
		Msg: &svcv1alpha1.CreateOrUpdateResourceResponse{
			Results: res,
		},
	}, nil
}

func (s *server) createOrUpdateResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) *svcv1alpha1.CreateOrUpdateResourceResult {
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), obj.DeepCopy()); err != nil {
		if kubeerr.IsNotFound(err) {
			// Create if resource not found
			switch res := s.createResource(ctx, obj).Result.(type) {
			case *svcv1alpha1.CreateResourceResult_CreatedResourceManifest:
				return &svcv1alpha1.CreateOrUpdateResourceResult{
					Result: &svcv1alpha1.CreateOrUpdateResourceResult_CreatedResourceManifest{
						CreatedResourceManifest: res.CreatedResourceManifest,
					},
				}
			case *svcv1alpha1.CreateResourceResult_Error:
				return newCreateOrUpdateResourceResultError(errors.New(res.Error))
			default:
				return newCreateOrUpdateResourceResultError(fmt.Errorf("unknown result type %T", res))
			}
		}
		return newCreateOrUpdateResourceResultError(err)
	}

	// Update if resource found
	switch res := s.updateResource(ctx, obj).Result.(type) {
	case *svcv1alpha1.UpdateResourceResult_UpdatedResourceManifest:
		return &svcv1alpha1.CreateOrUpdateResourceResult{
			Result: &svcv1alpha1.CreateOrUpdateResourceResult_UpdatedResourceManifest{
				UpdatedResourceManifest: res.UpdatedResourceManifest,
			},
		}
	case *svcv1alpha1.UpdateResourceResult_Error:
		return newCreateOrUpdateResourceResultError(errors.New(res.Error))
	default:
		return newCreateOrUpdateResourceResultError(fmt.Errorf("unknown result type %T", res))
	}
}

func newCreateOrUpdateResourceResultError(err error) *svcv1alpha1.CreateOrUpdateResourceResult {
	return &svcv1alpha1.CreateOrUpdateResourceResult{
		Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
			Error: fmt.Errorf("create or update resource: %w", err).Error(),
		},
	}
}
