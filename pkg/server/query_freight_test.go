package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestGetFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			server: &server{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight for Warehouse")
			},
		},
		{
			name: "success",
			server: &server{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.server.getFreightFromWarehouses(
				t.Context(),
				"fake-project",
				[]string{"fake-warehouse"},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetVerifiedFreight(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			server: &server{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight verified in Stage")
			},
		},
		{
			name: "success",
			server: &server{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				// Ensured the list is de-duped. If it weren't there would be 4 here.
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.server.getVerifiedFreight(
				t.Context(),
				"fake-project",
				[]string{
					"fake-stage",
					"another-fake-stage",
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func Test_server_queryFreight(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testFreight1 := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight-1",
			Namespace: testProject.Name,
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
		Images: []kargoapi.Image{
			{RepoURL: "example.com/image", Tag: "v1.0.0"},
		},
	}
	testFreight2 := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight-2",
			Namespace: testProject.Name,
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
		Commits: []kargoapi.GitCommit{
			{RepoURL: "https://github.com/example/repo", ID: "abc123"},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/freight",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "List all freight",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight1, testFreight2),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
			{
				name:          "Stage not found when filtering by stage",
				url:           "/v1beta1/projects/" + testProject.Name + "/freight?stage=nonexistent-stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Invalid groupBy value",
				url:           "/v1beta1/projects/" + testProject.Name + "/freight?groupBy=invalid",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Group by image repository",
				url:           "/v1beta1/projects/" + testProject.Name + "/freight?groupBy=image_repo",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight1),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
			{
				name:          "Group by git repository",
				url:           "/v1beta1/projects/" + testProject.Name + "/freight?groupBy=git_repo",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight2),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
		},
	)
}

func Test_server_queryFreight_watch(t *testing.T) {
	const projectName = "fake-project"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/freight?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/freight?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches all freight successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "freight-1",
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Create a new piece of freight to trigger a watch event
					newFreight := &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "freight-2",
						},
					}
					_ = c.Create(ctx, newFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the create operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
					require.Contains(t, body, "freight-2")
				},
			},
			{
				name: "filters watch events by origins",
				url:  "/v1beta1/projects/" + projectName + "/freight?watch=true&origins=warehouse-1",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Freight from the requested warehouse (should be sent)
					_ = c.Create(ctx, &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "freight-wh1",
						},
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "warehouse-1",
						},
					})
					// Freight from a different warehouse (should be filtered out)
					_ = c.Create(ctx, &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "freight-wh2",
						},
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "warehouse-2",
						},
					})
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

					body := w.Body.String()
					require.Contains(t, body, "freight-wh1")
					require.NotContains(t, body, "freight-wh2")
				},
			},
			{
				name: "watches empty freight list",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers are set
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
				},
			},
		},
	)
}
