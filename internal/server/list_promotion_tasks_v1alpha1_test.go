package server

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

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestListPromotionTasks(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListPromotionTasksRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListPromotionTasksResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListPromotionTasksRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionTasksResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing project": {
			req: &svcv1alpha1.ListPromotionTasksRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[kargoapi.PromotionTask]("testdata/promotion-task.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionTasksResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotionTasks(), 1)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListPromotionTasksRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionTasksResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by name": {
			req: &svcv1alpha1.ListPromotionTasksRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				func() client.Object {
					obj := mustNewObject[kargoapi.PromotionTask]("testdata/promotion-task.yaml")
					obj.SetName("z-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.PromotionTask]("testdata/promotion-task.yaml")
					obj.SetName("a-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.PromotionTask]("testdata/promotion-task.yaml")
					obj.SetName("m-promotion-task")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[kargoapi.PromotionTask]("testdata/promotion-task.yaml")
					obj.SetName("0-promotion-task")
					return obj
				}(),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionTasksResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotionTasks(), 4)

				// Check that the promotion tasks are ordered by name.
				require.Equal(t, "0-promotion-task", r.Msg.GetPromotionTasks()[0].GetName())
				require.Equal(t, "a-promotion-task", r.Msg.GetPromotionTasks()[1].GetName())
				require.Equal(t, "m-promotion-task", r.Msg.GetPromotionTasks()[2].GetName())
				require.Equal(t, "z-promotion-task", r.Msg.GetPromotionTasks()[3].GetName())
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
			res, err := (svr).ListPromotionTasks(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
