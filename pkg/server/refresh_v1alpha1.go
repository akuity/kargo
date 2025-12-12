package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/component"
)

// RefreshResourceType represents the type of Kargo resource to refresh.
// They are camel-cased versions of the Kargo resource kinds for compatibility
// purposes with Kubernetes REST mappers.
const (
	RefreshResourceTypeClusterConfig RefreshResourceType = "ClusterConfig"
	RefreshResourceTypeProjectConfig RefreshResourceType = "ProjectConfig"
	RefreshResourceTypeStage         RefreshResourceType = "Stage"
	RefreshResourceTypeWarehouse     RefreshResourceType = "Warehouse"
)

func init() {
	defaultRefreshObjectRegistry.MustRegister(
		refreshObjectRegistration{
			Predicate: func(_ context.Context, rt RefreshResourceType) (bool, error) {
				return RefreshResourceTypeClusterConfig.equals(rt), nil
			},
			Value: func() (client.Object, RefreshResourceType) {
				return new(kargoapi.ClusterConfig), RefreshResourceTypeClusterConfig
			},
		},
	)
	defaultRefreshObjectRegistry.MustRegister(
		refreshObjectRegistration{
			Predicate: func(_ context.Context, rt RefreshResourceType) (bool, error) {
				return RefreshResourceTypeProjectConfig.equals(rt), nil
			},
			Value: func() (client.Object, RefreshResourceType) {
				return new(kargoapi.ProjectConfig), RefreshResourceTypeProjectConfig
			},
		},
	)
	defaultRefreshObjectRegistry.MustRegister(
		refreshObjectRegistration{
			Predicate: func(_ context.Context, rt RefreshResourceType) (bool, error) {
				return RefreshResourceTypeStage.equals(rt), nil
			},
			Value: func() (client.Object, RefreshResourceType) {
				return new(kargoapi.Stage), RefreshResourceTypeStage
			},
		},
	)
	defaultRefreshObjectRegistry.MustRegister(
		refreshObjectRegistration{
			Predicate: func(_ context.Context, rt RefreshResourceType) (bool, error) {
				return RefreshResourceTypeWarehouse.equals(rt), nil
			},
			Value: func() (client.Object, RefreshResourceType) {
				return new(kargoapi.Warehouse), RefreshResourceTypeWarehouse
			},
		},
	)
}

type (
	refreshObjectPredicate = func(
		context.Context,
		RefreshResourceType,
	) (bool, error)

	refreshObjectFactory = func() (client.Object, RefreshResourceType)

	refreshObjectRegistration = component.PredicateBasedRegistration[
		RefreshResourceType,
		refreshObjectPredicate,
		refreshObjectFactory,
		struct{},
	]
)

var defaultRefreshObjectRegistry = component.MustNewPredicateBasedRegistry[
	RefreshResourceType,
	refreshObjectPredicate,
	refreshObjectFactory,
	struct{},
]()

type RefreshResourceType string

// String returns the string representation of the RefreshResourceType.
// If the RefreshResourceType is not registered, an empty string is returned.
func (t RefreshResourceType) String() string {
	registered, err := defaultRefreshObjectRegistry.Get(context.Background(), t)
	if err != nil {
		return ""
	}
	_, rt := registered.Value()
	return string(rt)
}

func (t RefreshResourceType) IsNamespaced() bool {
	return !t.equals(RefreshResourceTypeClusterConfig)
}

// NameEqualsProject returns true if the name of the resource should be the same
// as the project name. This is true for ProjectConfig resources.
func (t RefreshResourceType) NameEqualsProject() bool {
	return t.equals(RefreshResourceTypeProjectConfig)
}

// cli uses lowercase strings for resource types
func (t RefreshResourceType) equals(rt RefreshResourceType) bool {
	return strings.EqualFold(string(t), string(rt))
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
	rt := r.GetResourceType()
	registered, err := defaultRefreshObjectRegistry.Get(ctx, RefreshResourceType(rt))
	if err != nil {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("unsupported resource type %q: %w", rt, err),
		)
	}
	o, _ := registered.Value()
	o.SetNamespace(om.GetNamespace())
	o.SetName(om.GetName())
	return o, nil
}

func (s *server) getObjectMeta(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	rt, err := validateRefreshResourceType(ctx, r)
	if err != nil {
		return nil, err
	}
	if !rt.IsNamespaced() {
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
	if rt.NameEqualsProject() {
		o.SetName(r.GetProject())
		return &o, nil
	}
	if err := validateFieldNotEmpty("name", r.GetName()); err != nil {
		return nil, err
	}
	o.SetName(r.GetName())
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

func validateRefreshResourceType(
	ctx context.Context,
	r *svcv1alpha1.RefreshResourceRequest,
) (RefreshResourceType, error) {
	rt := RefreshResourceType(r.GetResourceType())
	if string(rt) == "" {
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("resource type is unset"),
		)
	}
	if _, err := defaultRefreshObjectRegistry.Get(ctx, rt); err != nil {
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("%q is unsupported as a refresh resource type: %w", rt, err),
		)
	}
	return rt, nil
}
