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

func (s *server) Refresh(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshRequest],
) (*connect.Response[svcv1alpha1.RefreshResponse], error) {
	o, err := s.getClientObject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), o); err != nil {
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

func (s *server) getClientObject(ctx context.Context, req *svcv1alpha1.RefreshRequest) (client.Object, error) {
	om, err := s.getObjectMeta(ctx, req)
	if err != nil {
		return nil, err
	}
	switch req.Kind {
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
			fmt.Errorf("unsupported refresh kind: %s", req.Kind),
		)
	}
}

func (s *server) getObjectMeta(ctx context.Context, req *svcv1alpha1.RefreshRequest) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	if req.Kind == "ClusterConfig" {
		o.SetName(api.ClusterConfigName)
		return &o, nil
	}

	if err := validateFieldNotEmpty("project", req.GetProject()); err != nil {
		return nil, err
	}
	if err := s.validateProjectExists(ctx, req.GetProject()); err != nil {
		return nil, err
	}
	o.SetNamespace(req.GetProject())

	if req.Kind != "ProjectConfig" {
		if err := validateFieldNotEmpty("name", req.GetName()); err != nil {
			return nil, err
		}
		o.SetName(req.GetName())
	} else {
		o.SetName(req.GetProject())
	}
	return &o, nil
}

func newRefreshResponse(obj client.Object) *connect.Response[svcv1alpha1.RefreshResponse] {
	b, _ := json.Marshal(obj)
	return connect.NewResponse(&svcv1alpha1.RefreshResponse{
		Object: &anypb.Any{
			TypeUrl: obj.GetObjectKind().GroupVersionKind().String(),
			Value:   b,
		},
	})
}
