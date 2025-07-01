package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestGetClusterAnalysisTemplate(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.GetClusterAnalysisTemplateRequest
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name: "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing ClusterAnalysisTemplate": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name: "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetClusterAnalysisTemplate())
				require.Equal(t, "test", c.Msg.GetClusterAnalysisTemplate().Name)
			},
		},
		"non-existing ClusterAnalysisTemplate": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name: "non-existing",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting ClusterAnalysisTemplate": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name: "test",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name: "test",
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name:   "test",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetClusterAnalysisTemplate())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.ClusterAnalysisTemplate)
				require.True(t, ok)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetClusterAnalysisTemplateRequest{
				Name:   "test",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetClusterAnalysisTemplate())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.ClusterAnalysisTemplate)
				require.True(t, ok)
				require.Equal(t, "test", tObj.Name)
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
								mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml"),
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
			res, err := (svr).GetClusterAnalysisTemplate(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
