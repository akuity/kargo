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
		return nil, err
	}

	if err = api.RefreshObject(ctx, s.client.InternalClient(), o); err != nil {
		return nil, err
	}

	// If we're dealing with a stage and there is a current promotion then refresh it too.
	stage, ok := o.(*kargoapi.Stage)
	if ok && stage.Status.CurrentPromotion != nil {
		err = api.RefreshObject(ctx, s.client.InternalClient(), &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: stage.Namespace,
				Name:      stage.Status.CurrentPromotion.Name,
			},
		})
		if err != nil {
			return nil, err
		}
	}
	return newRefreshResponse(o), nil
}

func (s *server) getClientObject(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (client.Object, error) {
	om, err := s.getObjectMeta(ctx, r)
	if err != nil {
		return nil, err
	}
	switch r.Kind {
	case "ClusterConfig":
		return &kargoapi.ClusterConfig{ObjectMeta: *om}, nil
	case "ProjectConfig":
		return &kargoapi.ProjectConfig{ObjectMeta: *om}, nil
	case "Warehouse":
		return &kargoapi.Warehouse{ObjectMeta: *om}, nil
	case "Stage":
		return &kargoapi.Stage{ObjectMeta: *om}, nil
	default:
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("unsupported refresh kind: %s", r.Kind),
		)
	}
}

func (s *server) getObjectMeta(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	if r.Kind == "ClusterConfig" {
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

	if r.Kind != "ProjectConfig" {
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
