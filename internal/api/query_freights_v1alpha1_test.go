package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestQueryFreights(t *testing.T) {
	testSets := map[string]struct {
		req                *svcv1alpha1.QueryFreightsRequest
		errMsg             string
		expectedCode       connect.Code
		expectedPromotions int32
		assertions         func(res *svcv1alpha1.QueryFreightsResponse)
	}{
		"empty project": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "",
			},
			errMsg:       "project should not be empty",
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing Stage": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				Stage:   "does-not-exist",
			},
			errMsg:       `stage "does-not-exist" not found`,
			expectedCode: connect.CodeNotFound,
		},
		"invalid group by": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				GroupBy: "notvalid",
			},
			errMsg:       `Invalid group by: notvalid`,
			expectedCode: connect.CodeInvalidArgument,
		},
		"invalid order by": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				OrderBy: "notvalid",
			},
			errMsg:       `Invalid order by: notvalid`,
			expectedCode: connect.CodeInvalidArgument,
		},
		"invalid group filter": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				Group:   "ghcr.io/akuity/guestbook",
			},
			errMsg:       "Cannot filter by group without group by",
			expectedCode: connect.CodeInvalidArgument,
		},
		"invalid order by tag": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				OrderBy: OrderByTag,
			},
			errMsg:       "Tag ordering only valid when grouping by: container_repo, helm_repo",
			expectedCode: connect.CodeInvalidArgument,
		},
		"query all freight": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 1)
				require.Len(t, res.GetGroups()[""].Freights, 4)
				require.Equal(t, res.GetGroups()[""].Freights[0].Id, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				require.Equal(t, res.GetGroups()[""].Freights[1].Id, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
				require.Equal(t, res.GetGroups()[""].Freights[2].Id, "cccccccccccccccccccccccccccccccccccccccc")
				require.Equal(t, res.GetGroups()[""].Freights[3].Id, "dddddddddddddddddddddddddddddddddddddddd")
			},
		},
		"reverse sort": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				Reverse: true,
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 1)
				require.Len(t, res.GetGroups()[""].Freights, 4)
				require.Equal(t, res.GetGroups()[""].Freights[0].Id, "dddddddddddddddddddddddddddddddddddddddd")
				require.Equal(t, res.GetGroups()[""].Freights[1].Id, "cccccccccccccccccccccccccccccccccccccccc")
				require.Equal(t, res.GetGroups()[""].Freights[2].Id, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
				require.Equal(t, res.GetGroups()[""].Freights[3].Id, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			},
		},
		"query single stage": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				Stage:   "query-freight-3",
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 1)
				require.Len(t, res.GetGroups()[""].Freights, 1)
				require.Equal(t, res.GetGroups()[""].Freights[0].Id, "dddddddddddddddddddddddddddddddddddddddd")
			},
		},
		"query group by container_repo": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				GroupBy: GroupByContainerRepository,
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 2)

				gb1 := res.GetGroups()["ghcr.io/akuity/guestbook"]
				require.Len(t, gb1.Freights, 3)
				require.Equal(t, gb1.Freights[0].Id, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				require.Equal(t, gb1.Freights[1].Id, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
				require.Equal(t, gb1.Freights[2].Id, "cccccccccccccccccccccccccccccccccccccccc")

				gb2 := res.GetGroups()["ghcr.io/akuity/guestbook2"]
				require.Len(t, gb2.Freights, 1)
				require.Equal(t, gb2.Freights[0].Id, "dddddddddddddddddddddddddddddddddddddddd")
			},
		},
		"group by container, order by tag": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				GroupBy: GroupByContainerRepository,
				OrderBy: OrderByTag,
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 2)

				gb1 := res.GetGroups()["ghcr.io/akuity/guestbook"]
				require.Len(t, gb1.Freights, 3)
				require.Equal(t, gb1.Freights[0].Images[0].Tag, "v0.0.1")
				require.Equal(t, gb1.Freights[1].Images[0].Tag, "v0.0.2")
				require.Equal(t, gb1.Freights[2].Images[0].Tag, "v0.0.4")

				gb2 := res.GetGroups()["ghcr.io/akuity/guestbook2"]
				require.Len(t, gb2.Freights, 1)
				require.Equal(t, gb2.Freights[0].Images[0].Tag, "v0.0.0")
			},
		},
		"filter by group": {
			req: &svcv1alpha1.QueryFreightsRequest{
				Project: "kargo-demo",
				GroupBy: GroupByContainerRepository,
				Group:   "ghcr.io/akuity/guestbook2",
			},
			assertions: func(res *svcv1alpha1.QueryFreightsResponse) {
				require.Len(t, res.GetGroups(), 1)
				gb2 := res.GetGroups()["ghcr.io/akuity/guestbook2"]
				require.Len(t, gb2.Freights, 1)
				require.Equal(t, gb2.Freights[0].Id, "dddddddddddddddddddddddddddddddddddddddd")
			},
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Simulate an admin user to prevent any authz issues with the authorizing
			// client.
			ctx := user.ContextWithInfo(
				context.Background(),
				user.Info{
					IsAdmin: true,
				},
			)

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (libClient.Client, error) {
						return fake.NewClientBuilder().
							WithScheme(mustNewScheme()).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
							).
							WithLists(&kubev1alpha1.StageList{
								Items: []kubev1alpha1.Stage{
									*mustNewObject[kubev1alpha1.Stage]("testdata/query-freight-1.yaml"),
									*mustNewObject[kubev1alpha1.Stage]("testdata/query-freight-2.yaml"),
									*mustNewObject[kubev1alpha1.Stage]("testdata/query-freight-3.yaml"),
								},
							}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			res, err := (&server{
				client: client,
			}).QueryFreights(ctx, connect.NewRequest(ts.req))
			if ts.errMsg != "" {
				require.ErrorContains(t, err, ts.errMsg)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			if ts.assertions != nil {
				ts.assertions(res.Msg)
			}

		})
	}
}
