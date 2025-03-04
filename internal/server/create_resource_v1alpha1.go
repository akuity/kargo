package server

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateResourceRequest],
) (*connect.Response[svcv1alpha1.CreateResourceResponse], error) {
	projects, otherResources, err := splitYAML(req.Msg.GetManifest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse manifest: %w", err))
	}
	resources := append(projects, otherResources...)
	results := make([]*svcv1alpha1.CreateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		result, err := s.createResource(ctx, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		results = append(results, result)
	}
	return &connect.Response[svcv1alpha1.CreateResourceResponse]{
		Msg: &svcv1alpha1.CreateResourceResponse{
			Results: results,
		},
	}, nil
}

func (s *server) createResource(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (*svcv1alpha1.CreateResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_Error{
				Error: errSecretManagementDisabled.Error(),
			},
		}, nil
	}

	// Note: We don't blindly attempt creating the resource because many resource
	// types have defaulting and/or validating webhooks and what we do not want is
	// for some error from a webhook to obscure the fact that the resource already
	// exists.
	existingObj := obj.DeepCopy()
	err := s.client.Get(ctx, client.ObjectKeyFromObject(obj), existingObj)
	if err == nil {
		// Whoops! This resource already exists. Make the error look just like we'd
		// gotten it from directly calling s.client.Create.
		err := kubeerr.NewAlreadyExists(
			schema.GroupResource{
				Group:    existingObj.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(existingObj.GetKind()),
			},
			existingObj.GetName(),
		)
		return &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_Error{
				Error: fmt.Errorf("create resource: %w", err).Error(),
			},
		}, err
	}
	if !kubeerr.IsNotFound(err) {
		return &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_Error{
				Error: fmt.Errorf("get resource: %w", err).Error(),
			},
		}, err
	}

	// If we get to here, the resource does not already exists, so we can create
	// it.

	if err = s.client.Create(ctx, obj); err != nil {
		return &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_Error{
				Error: fmt.Errorf("create resource: %w", err).Error(),
			},
		}, err
	}
	createdManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.CreateResourceResult{
			Result: &svcv1alpha1.CreateResourceResult_Error{
				Error: fmt.Errorf("marshal created manifest: %w", err).Error(),
			},
		}, err
	}
	return &svcv1alpha1.CreateResourceResult{
		Result: &svcv1alpha1.CreateResourceResult_CreatedResourceManifest{
			CreatedResourceManifest: createdManifest,
		},
	}, nil
}
