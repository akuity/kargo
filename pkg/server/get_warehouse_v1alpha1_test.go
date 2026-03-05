package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
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

			ctx := t.Context()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.WithWatch, error) {
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

func Test_prepareOutboundWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, *kargoapi.Warehouse, error)
	}{
		{
			name: "clears internal and creates external",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{
						{Git: &kargoapi.GitSubscription{}},
						{Image: &kargoapi.ImageSubscription{}},
						{Chart: &kargoapi.ChartSubscription{}},
						{Subscription: &kargoapi.Subscription{SubscriptionType: "generic"}},
					},
				},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.InternalSubscriptions))
				require.Greater(t, len(w.Spec.Subscriptions), 0)
			},
		},
		{
			name: "empty spec no changes",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.InternalSubscriptions))
				require.Equal(t, 0, len(w.Spec.Subscriptions))
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := prepareOutboundWarehouse(tc.warehouse)
			tc.assertions(t, tc.warehouse, err)
		})
	}
}

func Test_server_getWarehouse(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-warehouse",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/warehouses/"+testWarehouse.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Warehouse does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Warehouse",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Warehouse in the response
					warehouse := &kargoapi.Warehouse{}
					err := json.Unmarshal(w.Body.Bytes(), warehouse)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, warehouse.Namespace)
					require.Equal(t, testWarehouse.Name, warehouse.Name)
				},
			},
		},
	)
}

func Test_server_getWarehouse_watch(t *testing.T) {
	const projectName = "fake-project"
	const warehouseName = "fake-warehouse"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/warehouses/"+warehouseName+"?watch=true",
		[]restWatchTestCase{
			{
				name: "warehouse not found",
				url:  "/v1beta1/projects/" + projectName + "/warehouses/non-existent?watch=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches warehouse successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      warehouseName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Fetch the current warehouse first to get the resource version
					warehouse := &kargoapi.Warehouse{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: warehouseName}, warehouse)

					// Update the warehouse to trigger a watch event
					warehouse.Spec.FreightCreationPolicy = kargoapi.FreightCreationPolicyAutomatic
					_ = c.Update(ctx, warehouse)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the update operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
				},
			},
		},
	)
}
