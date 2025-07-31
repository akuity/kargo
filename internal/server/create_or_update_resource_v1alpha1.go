package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
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

	createdProjects := map[string]struct{}{}

	results := make([]*svcv1alpha1.CreateOrUpdateResourceResult, 0, len(resources))
	for _, r := range resources {
		resource := r // Avoid implicit memory aliasing
		var cl client.Client = s.client
		if _, ok := createdProjects[resource.GetNamespace()]; ok {
			// This resource belongs to a Project we created previously in this API
			// call. The user had sufficient permissions to accomplish that and having
			// done so makes them automatically the "owner" of the Project and an
			// admin. Most of those permissions are wrangled into place asynchronously
			// by the management controller, so in order to proceed with synchronously
			// creating resources within the Project at this time, we will use the API
			// server's own permissions to accomplish that. We accomplish that using
			// s.client's internal client for creation of this resource.
			cl = s.client.InternalClient()
		}
		result, err := s.createOrUpdateResource(ctx, cl, &resource)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if result.GetCreatedResourceManifest() != nil &&
			resource.GroupVersionKind() == projectGVK {
			createdProjects[resource.GetName()] = struct{}{}
		}
		results = append(results, result)
	}
	return &connect.Response[svcv1alpha1.CreateOrUpdateResourceResponse]{
		Msg: &svcv1alpha1.CreateOrUpdateResourceResponse{
			Results: results,
		},
	}, nil
}

func (s *server) createOrUpdateResource(
	ctx context.Context,
	cl client.Client,
	obj *unstructured.Unstructured,
) (*svcv1alpha1.CreateOrUpdateResourceResult, error) {
	if obj.GroupVersionKind() == secretGVK && !s.cfg.SecretManagementEnabled {
		return &svcv1alpha1.CreateOrUpdateResourceResult{
			Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
				Error: errSecretManagementDisabled.Error(),
			},
		}, nil
	}

	// Note: It would be tempting to blindly attempt creating the resource and
	// then update it instead if it already exists, but many resource types have
	// defaulting and/or validating webhooks and what we do not want is for some
	// error from a webhook to obscure the fact that the resource already exists.
	// So we'll explicitly check if the resource exists and then decide whether to
	// create or update it.
	existingObj := obj.DeepCopy()
	if err := cl.Get(ctx, client.ObjectKeyFromObject(obj), existingObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return &svcv1alpha1.CreateOrUpdateResourceResult{
				Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
					Error: fmt.Errorf("get resource: %w", err).Error(),
				},
			}, err
		}
		existingObj = nil
	}

	if existingObj == nil { // Create the resource
		// If the object is a Project, annotate it with information about the user
		// who created it.
		//
		// TODO(krancour): Do we want to do this for a broader variety of resource
		// types in the future?
		annotateProjectWithCreator(ctx, obj)
		if err := cl.Create(ctx, obj); err != nil {
			return &svcv1alpha1.CreateOrUpdateResourceResult{
				Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
					Error: fmt.Errorf("create resource: %w", err).Error(),
				},
			}, err
		}
		createdManifest, err := sigyaml.Marshal(obj)
		if err != nil {
			return &svcv1alpha1.CreateOrUpdateResourceResult{
				Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
					Error: fmt.Errorf("marshal created manifest: %w", err).Error(),
				},
			}, err
		}
		return &svcv1alpha1.CreateOrUpdateResourceResult{
			Result: &svcv1alpha1.CreateOrUpdateResourceResult_CreatedResourceManifest{
				CreatedResourceManifest: createdManifest,
			},
		}, nil
	}

	// If we get to here, the resource already exists, so we can update it.

	obj.SetResourceVersion(existingObj.GetResourceVersion())
	if err := cl.Update(ctx, obj); err != nil {
		return &svcv1alpha1.CreateOrUpdateResourceResult{
			Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
				Error: fmt.Errorf("update resource: %w", err).Error(),
			},
		}, err
	}
	updatedManifest, err := sigyaml.Marshal(obj)
	if err != nil {
		return &svcv1alpha1.CreateOrUpdateResourceResult{
			Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
				Error: fmt.Errorf("marshal updated manifest: %w", err).Error(),
			},
		}, err
	}
	return &svcv1alpha1.CreateOrUpdateResourceResult{
		Result: &svcv1alpha1.CreateOrUpdateResourceResult_UpdatedResourceManifest{
			UpdatedResourceManifest: updatedManifest,
		},
	}, nil
}
