package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) UpdateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateResourceRequest],
) (*connect.Response[svcv1alpha1.UpdateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(projects, otherResources...)
	results := make([]*svcv1alpha1.UpdateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		result, err := s.updateResource(ctx, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		results = append(results, result)
	}
	return &connect.Response[svcv1alpha1.UpdateResourceResponse]{
		Msg: &svcv1alpha1.UpdateResourceResponse{
			Results: results,
		},
	}, nil
}

func (s *server) updateResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (*svcv1alpha1.UpdateResourceResult, error) {
	// Note: We don't blindly attempt updating the resource because many resources
	// types have defaulting and/or validating webhooks and what we do not want is
	// for some error from a webhook to obscure the fact that the resource does
	// not exist.
	existingObj := obj.DeepCopy()
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), existingObj); err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: fmt.Errorf("get resource: %w", err).Error(),
			},
		}, err
	}

	// If we get to here, the resource already exists, so we can update it.

	obj.SetResourceVersion(existingObj.GetResourceVersion())
	if err := s.client.Update(ctx, obj); err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: fmt.Errorf("update resource: %w", err).Error(),
			},
		}, err
	}
	updatedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.UpdateResourceResult{
			Result: &svcv1alpha1.UpdateResourceResult_Error{
				Error: fmt.Errorf("marshal updated manifest: %w", err).Error(),
			},
		}, err
	}
	return &svcv1alpha1.UpdateResourceResult{
		Result: &svcv1alpha1.UpdateResourceResult_UpdatedResourceManifest{
			UpdatedResourceManifest: updatedManifest,
		},
	}, nil
}
