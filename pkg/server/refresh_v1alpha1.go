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
)

// RefreshResourceType represents the type of Kargo resource to refresh.
type RefreshResourceType string

// RefreshResourceType constants for supported resource types. They are
// PascalCase representations of the Kargo resource kinds for compatibility
// purposes with Kubernetes REST mappers.
const (
	RefreshResourceTypeClusterConfig RefreshResourceType = "ClusterConfig"
	RefreshResourceTypeProjectConfig RefreshResourceType = "ProjectConfig"
	RefreshResourceTypeStage         RefreshResourceType = "Stage"
	RefreshResourceTypeWarehouse     RefreshResourceType = "Warehouse"
)

// String returns the string representation of the RefreshResourceType.
func (t RefreshResourceType) String() string {
	return string(t)
}

// IsNamespaced returns true if the resource type is namespaced.
func (t RefreshResourceType) IsNamespaced() bool {
	return !strings.EqualFold(string(t), string(RefreshResourceTypeClusterConfig))
}

// NameEqualsProject returns true if the name of the resource should be the same
// as the project name. This is true for ProjectConfig resources.
func (t RefreshResourceType) NameEqualsProject() bool {
	return strings.EqualFold(string(t), string(RefreshResourceTypeProjectConfig))
}

// refreshObjectFactories maps resource types to their object factory functions.
var refreshObjectFactories = map[string]func() client.Object{
	strings.ToLower(string(RefreshResourceTypeClusterConfig)): func() client.Object { return &kargoapi.ClusterConfig{} },
	strings.ToLower(string(RefreshResourceTypeProjectConfig)): func() client.Object { return &kargoapi.ProjectConfig{} },
	strings.ToLower(string(RefreshResourceTypeStage)):         func() client.Object { return &kargoapi.Stage{} },
	strings.ToLower(string(RefreshResourceTypeWarehouse)):     func() client.Object { return &kargoapi.Warehouse{} },
}

func getRefreshObjectFactory(resourceType string) (func() client.Object, error) {
	factory, ok := refreshObjectFactories[strings.ToLower(resourceType)]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return factory, nil
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

	if err = s.authorizeRefresh(ctx, o); err != nil {
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

	// If we're dealing with a stage and there is a current promotion then
	// refresh it too.
	stage, ok := o.(*kargoapi.Stage)
	if ok && stage.Status.CurrentPromotion != nil {
		promo := &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: stage.Namespace,
				Name:      stage.Status.CurrentPromotion.Name,
			},
		}
		if err = s.authorizeRefresh(ctx, promo); err != nil {
			return nil, err
		}
		if err = api.RefreshObject(ctx, c, promo); err != nil {
			return nil, connect.NewError(
				connect.CodeInternal,
				fmt.Errorf("failed to refresh %s: %w", rt, err),
			)
		}
	}
	return newRefreshResponse(o), nil
}

// authorizeRefresh checks that the caller may "get" obj before it is
// refreshed. RefreshObject itself patches via the internal client, which
// performs no SubjectAccessReview, so this check is the only authorization
// this RPC performs.
func (s *server) authorizeRefresh(ctx context.Context, obj client.Object) error {
	var resource string
	switch obj.(type) {
	case *kargoapi.ClusterConfig:
		resource = "clusterconfigs"
	case *kargoapi.ProjectConfig:
		resource = "projectconfigs"
	case *kargoapi.Stage:
		resource = "stages"
	case *kargoapi.Warehouse:
		resource = "warehouses"
	case *kargoapi.Promotion:
		resource = promotionsResource
	default:
		return fmt.Errorf("unsupported type for refresh authorization: %T", obj)
	}
	return s.authorizeFn(
		ctx, "get", kargoapi.GroupVersion.WithResource(resource), "", client.ObjectKeyFromObject(obj),
	)
}

func (s *server) getClientObject(ctx context.Context, r *svcv1alpha1.RefreshResourceRequest) (client.Object, error) {
	om, err := s.getObjectMeta(ctx, r)
	if err != nil {
		return nil, err
	}
	factory, err := getRefreshObjectFactory(r.GetResourceType())
	if err != nil {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("%q is unsupported as a refresh resource type", r.GetResourceType()),
		)
	}
	o := factory()
	o.SetNamespace(om.GetNamespace())
	o.SetName(om.GetName())
	return o, nil
}

func (s *server) getObjectMeta(
	ctx context.Context,
	r *svcv1alpha1.RefreshResourceRequest,
) (*metav1.ObjectMeta, error) {
	var o metav1.ObjectMeta
	rt, err := validateRefreshResourceType(r)
	if err != nil {
		return nil, err
	}
	if !rt.IsNamespaced() {
		o.SetName(api.ClusterConfigName)
		return &o, nil
	}
	if err = validateFieldNotEmpty("project", r.GetProject()); err != nil {
		return nil, err
	}
	if err = s.validateProjectExists(ctx, r.GetProject()); err != nil {
		return nil, err
	}
	o.SetNamespace(r.GetProject())
	if rt.NameEqualsProject() {
		o.SetName(r.GetProject())
		return &o, nil
	}
	if err = validateFieldNotEmpty("name", r.GetName()); err != nil {
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

func validateRefreshResourceType(r *svcv1alpha1.RefreshResourceRequest) (RefreshResourceType, error) {
	rt := r.GetResourceType()
	if rt == "" {
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("resource type is unset"),
		)
	}
	if _, err := getRefreshObjectFactory(rt); err != nil {
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("%q is unsupported as a refresh resource type", rt),
		)
	}
	return RefreshResourceType(rt), nil
}
