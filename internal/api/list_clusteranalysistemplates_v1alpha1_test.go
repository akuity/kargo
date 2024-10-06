package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
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

func TestListClusterAnalysisTemplates(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListClusterAnalysisTemplatesRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], error)
	}{
		"existing": {
			req: &svcv1alpha1.ListClusterAnalysisTemplatesRequest{},
			objects: []client.Object{
				mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetClusterAnalysisTemplates(), 1)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req:              &svcv1alpha1.ListClusterAnalysisTemplatesRequest{},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by name": {
			req: &svcv1alpha1.ListClusterAnalysisTemplatesRequest{},
			objects: []client.Object{
				func() client.Object {
					obj := mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml")
					obj.SetName("z-clusteranalysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml")
					obj.SetName("a-clusteranalysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml")
					obj.SetName("m-clusteranalysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml")
					obj.SetName("0-clusteranalysistemplate")
					return obj
				}(),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetClusterAnalysisTemplates(), 4)

				// Check that the analysis templates are ordered by name.
				require.Equal(t, "0-clusteranalysistemplate", r.Msg.GetClusterAnalysisTemplates()[0].GetName())
				require.Equal(t, "a-clusteranalysistemplate", r.Msg.GetClusterAnalysisTemplates()[1].GetName())
				require.Equal(t, "m-clusteranalysistemplate", r.Msg.GetClusterAnalysisTemplates()[2].GetName())
				require.Equal(t, "z-clusteranalysistemplate", r.Msg.GetClusterAnalysisTemplates()[3].GetName())
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
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
			res, err := (svr).ListClusterAnalysisTemplates(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
