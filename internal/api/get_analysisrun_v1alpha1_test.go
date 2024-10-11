package api

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

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetAnalysisRun(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.GetAnalysisRunRequest
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.GetAnalysisRunResponse], error)
	}{
		"empty namespace": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "",
				Name:      "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing AnalysisRun": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetAnalysisRun())
				require.Equal(t, "kargo-demo", c.Msg.GetAnalysisRun().Namespace)
				require.Equal(t, "test", c.Msg.GetAnalysisRun().Name)
			},
		},
		"non-existing namespace": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-x",
				Name:      "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing AnalysisRun": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "non-existing",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting AnalysisRun": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
				Format:    svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetAnalysisRun())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.AnalysisRun)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
				Format:    svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisRunResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetAnalysisRun())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.AnalysisRun)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
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

			c, err := kubernetes.NewClient(
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
								mustNewObject[rollouts.AnalysisRun]("testdata/analysisrun.yaml"),
							).
							WithInterceptorFuncs(testCase.interceptor).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				cfg:    cfg,
				client: c,
			}
			res, err := (svr).GetAnalysisRun(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
