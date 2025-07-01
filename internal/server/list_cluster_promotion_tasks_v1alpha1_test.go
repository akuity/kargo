package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestListClusterPromotionTasks(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListClusterPromotionTasksRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListClusterPromotionTasksResponse], error)
	}{
		"existing": {
			req: &svcv1alpha1.ListClusterPromotionTasksRequest{},
			objects: []client.Object{
				mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListClusterPromotionTasksResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetClusterPromotionTasks(), 1)
			},
		},
		"orders by name": {
			req: &svcv1alpha1.ListClusterPromotionTasksRequest{},
			objects: []client.Object{
				func() client.Object {
					obj := mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml")
					obj.SetName("z-cluster-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml")
					obj.SetName("a-cluster-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml")
					obj.SetName("m-cluster-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.ClusterPromotionTask]("testdata/cluster-promotion-task.yaml")
					obj.SetName("0-cluster-promotion-task")
					return obj
				}(),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListClusterPromotionTasksResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetClusterPromotionTasks(), 4)

				// Check that the promotion tasks are ordered by name.
				require.Equal(t, "0-cluster-promotion-task", r.Msg.GetClusterPromotionTasks()[0].GetName())
				require.Equal(t, "a-cluster-promotion-task", r.Msg.GetClusterPromotionTasks()[1].GetName())
				require.Equal(t, "m-cluster-promotion-task", r.Msg.GetClusterPromotionTasks()[2].GetName())
				require.Equal(t, "z-cluster-promotion-task", r.Msg.GetClusterPromotionTasks()[3].GetName())
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
			res, err := (svr).ListClusterPromotionTasks(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
