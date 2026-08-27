package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

// createOrUpdateResourceResponse is the response for creating or updating resources
type createOrUpdateResourceResponse struct {
	Results []createOrUpdateResourceResult `json:"results"`
} // @name CreateOrUpdateResourceResponse

// createOrUpdateResourceResult is the result of creating or updating a resource
type createOrUpdateResourceResult struct {
	CreatedResourceManifest map[string]any `json:"createdResourceManifest,omitempty"`
	UpdatedResourceManifest map[string]any `json:"updatedResourceManifest,omitempty"`
	Error                   string         `json:"error,omitempty"`
} // @name CreateOrUpdateResourceResult

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
			// Kubernetes ignores metadata.namespace for a cluster-scoped kind, so a
			// spoofed value could bypass authorization there. Confirm real scope via
			// the RESTMapper before trusting it.
			if namespaced, err := s.client.IsObjectNamespaced(&resource); err == nil && namespaced {
				// Creating the Project made the user its owner/admin, but RBAC for it
				// is wired up asynchronously by the management controller. Use the API
				// server's own permissions here so this isn't blocked by that race.
				cl = s.client.InternalClient()
			}
		}
		result, err := s.updateResource(ctx, cl, &resource, true)
		if err != nil && len(resources) == 1 {
			return nil, err
		}
		// If we just created a Project successfully, keep track of this Project
		// being one that was created in the course of this API call.
		if err == nil && result.CreatedResourceManifest != nil && resource.GroupVersionKind() == projectGVK {
			createdProjects[resource.GetName()] = struct{}{}
		}
		// Convert to protobuf result
		var protoResult *svcv1alpha1.CreateOrUpdateResourceResult
		if result.Error != "" {
			protoResult = &svcv1alpha1.CreateOrUpdateResourceResult{
				Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
					Error: result.Error,
				},
			}
		} else if result.CreatedResourceManifest != nil {
			manifestBytes, marshalErr := sigyaml.Marshal(result.CreatedResourceManifest)
			if marshalErr != nil {
				protoResult = &svcv1alpha1.CreateOrUpdateResourceResult{
					Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
						Error: fmt.Errorf("marshal created manifest: %w", marshalErr).Error(),
					},
				}
			} else {
				protoResult = &svcv1alpha1.CreateOrUpdateResourceResult{
					Result: &svcv1alpha1.CreateOrUpdateResourceResult_CreatedResourceManifest{
						CreatedResourceManifest: manifestBytes,
					},
				}
			}
		} else {
			manifestBytes, marshalErr := sigyaml.Marshal(result.UpdatedResourceManifest)
			if marshalErr != nil {
				protoResult = &svcv1alpha1.CreateOrUpdateResourceResult{
					Result: &svcv1alpha1.CreateOrUpdateResourceResult_Error{
						Error: fmt.Errorf("marshal updated manifest: %w", marshalErr).Error(),
					},
				}
			} else {
				protoResult = &svcv1alpha1.CreateOrUpdateResourceResult{
					Result: &svcv1alpha1.CreateOrUpdateResourceResult_UpdatedResourceManifest{
						UpdatedResourceManifest: manifestBytes,
					},
				}
			}
		}
		results = append(results, protoResult)
	}
	return &connect.Response[svcv1alpha1.CreateOrUpdateResourceResponse]{
		Msg: &svcv1alpha1.CreateOrUpdateResourceResponse{Results: results},
	}, nil
}
