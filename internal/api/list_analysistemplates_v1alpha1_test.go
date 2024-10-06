package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListAnalysisTemplates(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListAnalysisTemplatesRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetAnalysisTemplates(), 1)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by name": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("z-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("a-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("m-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("0-analysistemplate")
					return obj
				}(),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetAnalysisTemplates(), 4)

				// Check that the analysis templates are ordered by name.
				require.Equal(t, "0-analysistemplate", r.Msg.GetAnalysisTemplates()[0].GetName())
				require.Equal(t, "a-analysistemplate", r.Msg.GetAnalysisTemplates()[1].GetName())
				require.Equal(t, "m-analysistemplate", r.Msg.GetAnalysisTemplates()[2].GetName())
				require.Equal(t, "z-analysistemplate", r.Msg.GetAnalysisTemplates()[3].GetName())
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
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				cfg:                       cfg,
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := (svr).ListAnalysisTemplates(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
