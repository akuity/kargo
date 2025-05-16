package server

import (
	"context"
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
	"github.com/akuity/kargo/internal/server/kubernetes"
)

func TestListProjects(t *testing.T) {
	testCases := map[string]struct {
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.ListProjectsResponse], error)
	}{
		"no projects": {
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Empty(t, r.Msg.GetProjects())
			},
		},
		"orders by name": {
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "z-project",
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "a-project",
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "m-project",
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "0-project",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetProjects(), 4)

				// Check that the projects are ordered by name.
				require.Equal(t, "0-project", r.Msg.GetProjects()[0].GetName())
				require.Equal(t, "a-project", r.Msg.GetProjects()[1].GetName())
				require.Equal(t, "m-project", r.Msg.GetProjects()[2].GetName())
				require.Equal(t, "z-project", r.Msg.GetProjects()[3].GetName())
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
				client: client,
			}
			res, err := (svr).ListProjects(ctx, &connect.Request[svcv1alpha1.ListProjectsRequest]{
				Msg: &svcv1alpha1.ListProjectsRequest{},
			})
			testCase.assertions(t, res, err)
		})
	}
}
