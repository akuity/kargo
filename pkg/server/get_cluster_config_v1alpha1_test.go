package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestGetClusterConfig(t *testing.T) {
	testCases := map[string]struct {
		req         *svcv1alpha1.GetClusterConfigRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetClusterConfigResponse], error)
	}{
		"non-existing ClusterConfig": {
			req:     &svcv1alpha1.GetClusterConfigRequest{},
			objects: []client.Object{},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetClusterConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ClusterConfig": {
			req: &svcv1alpha1.GetClusterConfigRequest{},
			objects: []client.Object{
				&kargoapi.ClusterConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: api.ClusterConfigName,
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetClusterConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetRaw())

				require.NotNil(t, r.Msg.GetClusterConfig())
				require.Equal(t, api.ClusterConfigName, r.Msg.GetClusterConfig().Name)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetClusterConfigRequest{
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				&kargoapi.ClusterConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: api.ClusterConfigName,
					},
					Spec: kargoapi.ClusterConfigSpec{
						WebhookReceivers: []kargoapi.WebhookReceiverConfig{
							{
								Name: "my-webhook-receiver",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetClusterConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetClusterConfig())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)

				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ClusterConfig)
				require.True(t, ok)
				require.Equal(t, api.ClusterConfigName, tObj.Name)
				require.Equal(t, 1, len(tObj.Spec.WebhookReceivers))
				require.Equal(t, "my-webhook-receiver", tObj.Spec.WebhookReceivers[0].Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetClusterConfigRequest{
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				&kargoapi.ClusterConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: api.ClusterConfigName,
					},
					Spec: kargoapi.ClusterConfigSpec{
						WebhookReceivers: []kargoapi.WebhookReceiverConfig{
							{
								Name: "my-webhook-receiver",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetClusterConfig())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ClusterConfig)
				require.True(t, ok)
				require.Equal(t, api.ClusterConfigName, tObj.Name)
				require.Equal(t, 1, len(tObj.Spec.WebhookReceivers))
				require.Equal(t, "my-webhook-receiver", tObj.Spec.WebhookReceivers[0].Name)
			},
		},
	}

	for name, testCase := range testCases {
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
						c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(testCase.interceptor)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
			}

			res, err := (svr).GetClusterConfig(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_getClusterConfig(t *testing.T) {
	testConfig := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/cluster-config",
		[]restTestCase{
			{
				name: "ClusterConfig does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ClusterConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfig),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterConfig in the response
					config := &kargoapi.ClusterConfig{}
					err := json.Unmarshal(w.Body.Bytes(), config)
					require.NoError(t, err)
					require.Equal(t, api.ClusterConfigName, config.Name)
				},
			},
		},
	)
}

func Test_server_getClusterConfig_watch(t *testing.T) {
	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/system/cluster-config?watch=true",
		[]restWatchTestCase{
			{
				name:          "cluster config not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches cluster config successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Update the cluster config to trigger a watch event
					// Fetch the current config first to get the resource version
					config := &kargoapi.ClusterConfig{}
					_ = c.Get(ctx, client.ObjectKey{Name: api.ClusterConfigName}, config)

					config.Spec.WebhookReceivers = []kargoapi.WebhookReceiverConfig{
						{Name: "new-receiver"},
					}
					_ = c.Update(ctx, config)
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
