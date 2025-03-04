package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetWarehouse(t *testing.T) {
	testSets := map[string]struct {
		req         *svcv1alpha1.GetWarehouseRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetWarehouseResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing Warehouse": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetWarehouse())
				require.Equal(t, "kargo-demo", c.Msg.GetWarehouse().Namespace)
				require.Equal(t, "test", c.Msg.GetWarehouse().Name)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing Warehouse": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "non-existing",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting Warehouse": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			interceptor: interceptor.Funcs{
				// This interceptor will be called when the client.Get method is called.
				// It will return an error to simulate a failure in the client.Get method.
				Get: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectKey,
					_ client.Object,
					_ ...client.GetOption,
				) error {
					return apierrors.NewServiceUnavailable("test")
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Warehouse{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Warehouse",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetWarehouse())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Warehouse)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetWarehouseRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Warehouse{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Warehouse",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetWarehouseResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetWarehouse())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Warehouse)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)

			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(ts.interceptor)
						if ts.objects != nil {
							c.WithObjects(ts.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := (svr).GetWarehouse(ctx, connect.NewRequest(ts.req))
			ts.assertions(t, res, err)
		})
	}
}
