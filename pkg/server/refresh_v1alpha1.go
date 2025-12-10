package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

const (
	RefreshResourceTypeClusterConfig RefreshResourceType = "clusterconfig"
	RefreshResourceTypeProjectConfig RefreshResourceType = "projectconfig"
	RefreshResourceTypeStage         RefreshResourceType = "stage"
	RefreshResourceTypeWarehouse     RefreshResourceType = "warehouse"
)

var ObjectRefreshResourceType = map[RefreshResourceType]client.Object{
	RefreshResourceTypeClusterConfig: new(kargoapi.ClusterConfig),
	RefreshResourceTypeProjectConfig: new(kargoapi.ProjectConfig),
	RefreshResourceTypeStage:         new(kargoapi.Stage),
	RefreshResourceTypeWarehouse:     new(kargoapi.Warehouse),
}

type RefreshResourceType string

func (t RefreshResourceType) String() string {
	// normalize for type-casts
	s := strings.ToLower(string(t))
	return strings.TrimSpace(s)
}

func (t RefreshResourceType) RequiresName() bool {
	switch t {
	case RefreshResourceTypeStage, RefreshResourceTypeWarehouse:
		return true
	default:
		return false
	}
}

func (t RefreshResourceType) RequiresProject() bool {
	switch t {
	case RefreshResourceTypeProjectConfig, RefreshResourceTypeStage, RefreshResourceTypeWarehouse:
		return true
	default:
		return false
	}
}

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
	rt := RefreshResourceType(r.GetResourceType())
	object, ok := ObjectRefreshResourceType[rt]
	if !ok {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("unsupported refresh kind: %s", rt),
		)
	}
	object.SetNamespace(om.GetNamespace())
	object.SetName(om.GetName())
	return object, nil
}

func (s *server) getObjectMeta(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	rt := RefreshResourceType(r.GetResourceType())
	if !rt.RequiresName() && !rt.RequiresProject() {
		o.SetName(api.ClusterConfigName)
		return &o, nil
	}

	if rt.RequiresProject() {
		if err := validateFieldNotEmpty("project", r.GetProject()); err != nil {
			return nil, err
		}
		if err := s.validateProjectExists(ctx, r.GetProject()); err != nil {
			return nil, err
		}
		o.SetNamespace(r.GetProject())
	}

	if rt.RequiresName() {
		if err := validateFieldNotEmpty("name", r.GetName()); err != nil {
			return nil, err
		}
		o.SetName(r.GetName())
		return &o, nil
	}
	o.SetName(r.GetProject())
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
