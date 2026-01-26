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

			ctx := t.Context()

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
				client: client,
			}
			res, err := (svr).ListProjects(ctx, &connect.Request[svcv1alpha1.ListProjectsRequest]{
				Msg: &svcv1alpha1.ListProjectsRequest{},
			})
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_listProjects(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects",
		[]restTestCase{
			{
				name: "no Projects exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists Projects",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "z-project"}},
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "a-project"}},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Projects in the response
					projects := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), projects)
					require.NoError(t, err)
					require.Len(t, projects.Items, 2)
					require.Equal(t, "a-project", projects.Items[0].Name)
					require.Equal(t, "z-project", projects.Items[1].Name)
				},
			},
		},
	)
}
