package server

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func (s *server) RefreshResource(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshResourceRequest],
) (*connect.Response[svcv1alpha1.RefreshResourceResponse], error) {
	o, err := s.getClientObject(ctx, req.Msg)
	if err != nil {
		// errors returned here are already properly formed connect errors
		return nil, err
	}

	c := s.client.InternalClient()
	rt := req.Msg.GetResourceType()

	if err = api.RefreshObject(ctx, c, o); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("%s not found", rt),
			)
		}
		return nil, connect.NewError(
			connect.CodeInternal,
			fmt.Errorf("failed to refresh %s: %w", rt, err),
		)
	}

	// If we're dealing with a stage and there is a current promotion then refresh it too.
	stage, ok := o.(*kargoapi.Stage)
	if ok && stage.Status.CurrentPromotion != nil {
		err = api.RefreshObject(ctx, c, &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: stage.Namespace,
				Name:      stage.Status.CurrentPromotion.Name,
			},
		})
		if err != nil {
			return nil, connect.NewError(
				connect.CodeInternal,
				fmt.Errorf("failed to refresh %s: %w", rt, err),
			)
		}
	}
	return newRefreshResponse(o), nil
}

func (s *server) getClientObject(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (client.Object, error) {
	om, err := s.getObjectMeta(ctx, r)
	if err != nil {
		return nil, err
	}
	switch rt := r.GetResourceType(); rt {
	case svcv1alpha1.RefreshResourceType_CLUSTER_CONFIG:
		return &kargoapi.ClusterConfig{ObjectMeta: *om}, nil
	case svcv1alpha1.RefreshResourceType_PROJECT_CONFIG:
		return &kargoapi.ProjectConfig{ObjectMeta: *om}, nil
	case svcv1alpha1.RefreshResourceType_WAREHOUSE:
		return &kargoapi.Warehouse{ObjectMeta: *om}, nil
	case svcv1alpha1.RefreshResourceType_STAGE:
		return &kargoapi.Stage{ObjectMeta: *om}, nil
	default:
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("unsupported refresh kind: %s", rt),
		)
	}
}

func (s *server) getObjectMeta(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	if r.ResourceType == svcv1alpha1.RefreshResourceType_CLUSTER_CONFIG {
		o.SetName(api.ClusterConfigName)
		return &o, nil
	}

	if err := validateFieldNotEmpty("project", r.GetProject()); err != nil {
		return nil, err
	}
	if err := s.validateProjectExists(ctx, r.GetProject()); err != nil {
		return nil, err
	}
	o.SetNamespace(r.GetProject())

	if r.ResourceType != svcv1alpha1.RefreshResourceType_PROJECT_CONFIG {
		if err := validateFieldNotEmpty("name", r.GetName()); err != nil {
			return nil, err
		}
		o.SetName(r.GetName())
	} else {
		o.SetName(r.GetProject())
	}
	return &o, nil
}

func newRefreshResponse(obj client.Object) *connect.Response[svcv1alpha1.RefreshResourceResponse] {
	b, _ := json.Marshal(obj)
	return connect.NewResponse(&svcv1alpha1.RefreshResourceResponse{
		Resource: &anypb.Any{
			TypeUrl: obj.GetObjectKind().GroupVersionKind().String(),
			Value:   b,
		},
	})
}
