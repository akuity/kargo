package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestGetClusterPromotionTask(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.GetClusterPromotionTaskRequest
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name: "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing ClusterPromotionTask": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name: "global-task",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetPromotionTask())
				require.Equal(t, "global-task", c.Msg.GetPromotionTask().Name)
			},
		},
		"non-existing ClusterPromotionTask": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name: "non-existing-global-task",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting ClusterPromotionTask": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name: "global-task",
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
					return apierrors.NewServiceUnavailable("global-task")
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name:   "global-task",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetPromotionTask())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ClusterPromotionTask)
				require.True(t, ok)
				require.Equal(t, "global-task", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetClusterPromotionTaskRequest{
				Name:   "global-task",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterPromotionTaskResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetPromotionTask())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ClusterPromotionTask)
				require.True(t, ok)
				require.Equal(t, "global-task", tObj.Name)
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			cfg := config.ServerConfigFromEnv()
			if testCase.rolloutsDisabled {
				cfg.RolloutsIntegrationEnabled = false
			}

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
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml"),
							).
							WithInterceptorFuncs(testCase.interceptor).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				cfg:                       cfg,
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := (svr).GetClusterPromotionTask(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
