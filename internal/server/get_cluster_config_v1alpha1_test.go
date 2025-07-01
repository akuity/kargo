package server

import (
	"context"
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
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/server/kubernetes"
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
