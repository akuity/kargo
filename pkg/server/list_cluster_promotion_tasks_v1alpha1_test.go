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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
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

			ctx := t.Context()

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
					) (client.WithWatch, error) {
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

func Test_server_listClusterPromotionTasks(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/shared/cluster-promotion-tasks",
		[]restTestCase{
			{
				name: "no ClusterPromotionTasks exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.ClusterPromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ClusterPromotionTasks",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.ClusterPromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Name: "task-1",
						},
					},
					&kargoapi.ClusterPromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Name: "task-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterPromotionTasks in the response
					tasks := &kargoapi.ClusterPromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), tasks)
					require.NoError(t, err)
					require.Len(t, tasks.Items, 2)
				},
			},
		},
	)
}
