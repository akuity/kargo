package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

func TestListStages(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.ListStagesRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.ListStagesRequest{
				Project: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing project": {
			req: &svcv1alpha1.ListStagesRequest{
				Project: "kargo-demo",
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListStagesRequest{
				Project: "non-existing-project",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (client.WithWatch, error) {
						return fake.NewClientBuilder().
							WithScheme(mustNewScheme()).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
							).
							WithLists(&kargoapi.StageList{
								Items: []kargoapi.Stage{
									*mustNewObject[kargoapi.Stage]("testdata/stage.yaml"),
								},
							}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := (svr).ListStages(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.Len(t, res.Msg.GetStages(), 1)
		})
	}
}

func Test_server_listStages(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/stages",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Stages exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					err := json.Unmarshal(w.Body.Bytes(), stages)
					require.NoError(t, err)
					require.Empty(t, stages.Items)
				},
			},
			{
				name: "lists Stages",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "stage-1",
						},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "stage-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Stages in the response
					stages := &kargoapi.StageList{}
					err := json.Unmarshal(w.Body.Bytes(), stages)
					require.NoError(t, err)
					require.Len(t, stages.Items, 2)
				},
			},
		},
	)
}

func Test_server_listStages_watch(t *testing.T) {
	const projectName = "fake-project"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/stages?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/stages?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches all stages successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-1",
						},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-2",
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Create a new stage to trigger a watch event
					newStage := &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-3",
						},
					}
					_ = c.Create(ctx, newStage)
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
				},
			},
			{
				name: "watches empty stage list",
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
