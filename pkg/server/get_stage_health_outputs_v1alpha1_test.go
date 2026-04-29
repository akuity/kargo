package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

func TestGetStageHealthOutputs(t *testing.T) {
	const projectName = "kargo-demo"

	stageWithOutput := func(name, raw string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: name},
			Status: kargoapi.StageStatus{
				Health: &kargoapi.Health{
					Status: kargoapi.HealthStateHealthy,
					Output: &apiextensionsv1.JSON{Raw: []byte(raw)},
				},
			},
		}
	}
	stageHealthNoOutput := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: "no-output"},
		Status: kargoapi.StageStatus{
			Health: &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
		},
	}
	stageNoHealth := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: "no-health"},
	}

	testCases := map[string]struct {
		req    *svcv1alpha1.GetStageHealthOutputsRequest
		assert func(*testing.T, *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{Project: ""},
			assert: func(t *testing.T, _ *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{Project: "does-not-exist"},
			assert: func(t *testing.T, _ *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			},
		},
		"empty stage_names returns empty map, not error": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{Project: projectName},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, res.Msg.GetHealthOutputs())
			},
		},
		"all empty strings treated as empty": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"", ""},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, res.Msg.GetHealthOutputs())
			},
		},
		"returns outputs for existing stages with health output": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"a", "b"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				got := res.Msg.GetHealthOutputs()
				require.Len(t, got, 2)
				require.Equal(t, `{"app":"a"}`, got["a"])
				require.Equal(t, `{"app":"b"}`, got["b"])
			},
		},
		"unknown stage names are silently omitted": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"a", "unknown"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				got := res.Msg.GetHealthOutputs()
				require.Len(t, got, 1)
				require.Contains(t, got, "a")
				require.NotContains(t, got, "unknown")
			},
		},
		"stage with health but no output is omitted": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"no-output"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, res.Msg.GetHealthOutputs())
			},
		},
		"stage with no health is omitted": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"no-health"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, res.Msg.GetHealthOutputs())
			},
		},
		"duplicate stage names are deduplicated": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: []string{"a", "a", "a"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.NoError(t, err)
				got := res.Msg.GetHealthOutputs()
				require.Len(t, got, 1)
				require.Equal(t, `{"app":"a"}`, got["a"])
			},
		},
		"batch size cap exceeded": {
			req: &svcv1alpha1.GetStageHealthOutputsRequest{
				Project:    projectName,
				StageNames: manyNames(maxStageHealthOutputsBatch + 1),
			},
			assert: func(t *testing.T, _ *connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			kubeClient, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context, *rest.Config, *runtime.Scheme,
					) (client.WithWatch, error) {
						return fake.NewClientBuilder().
							WithScheme(mustNewScheme()).
							WithObjects(
								&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
									Name:   projectName,
									Labels: map[string]string{kargoapi.LabelKeyProject: kargoapi.LabelValueTrue},
								}},
								&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}},
							).
							WithLists(&kargoapi.StageList{Items: []kargoapi.Stage{
								*stageWithOutput("a", `{"app":"a"}`),
								*stageWithOutput("b", `{"app":"b"}`),
								*stageHealthNoOutput,
								*stageNoHealth,
							}}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{client: kubeClient}
			svr.externalValidateProjectFn = validation.ValidateProject

			res, err := svr.GetStageHealthOutputs(ctx, connect.NewRequest(tc.req))
			tc.assert(t, res, err)
		})
	}
}

func manyNames(n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = fmt.Sprintf("stage-%d", i)
	}
	return out
}

func manyStageNamesQuery(n int) string {
	parts := make([]string, n)
	for i := range n {
		parts[i] = "stageNames=stage-" + strconv.Itoa(i)
	}
	return strings.Join(parts, "&")
}

func Test_server_getStageHealthOutputs(t *testing.T) {
	testProject := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}}

	stageWithOutput := func(name, raw string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: name},
			Status: kargoapi.StageStatus{
				Health: &kargoapi.Health{
					Status: kargoapi.HealthStateHealthy,
					Output: &apiextensionsv1.JSON{Raw: []byte(raw)},
				},
			},
		}
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/stage-health-outputs",
		[]restTestCase{
			{
				name: "project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "empty stageNames returns empty map",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					resp := &svcv1alpha1.GetStageHealthOutputsResponse{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
					require.Empty(t, resp.GetHealthOutputs())
				},
			},
			{
				name: "returns only requested stages that have output",
				url:  "/v1beta1/projects/" + testProject.Name + "/stage-health-outputs?stageNames=a&stageNames=c",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageWithOutput("a", `{"app":"a"}`),
					stageWithOutput("b", `{"app":"b"}`),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					resp := &svcv1alpha1.GetStageHealthOutputsResponse{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
					got := resp.GetHealthOutputs()
					require.Len(t, got, 1)
					require.Equal(t, `{"app":"a"}`, got["a"])
					require.NotContains(t, got, "b")
					require.NotContains(t, got, "c")

					// Lock in the wire shape: each value must be a JSON
					// string, not a base64-encoded blob or array of ints.
					var raw struct {
						HealthOutputs map[string]json.RawMessage `json:"health_outputs"`
					}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
					require.JSONEq(
						t,
						`"{\"app\":\"a\"}"`,
						string(raw.HealthOutputs["a"]),
					)
				},
			},
			{
				name: "batch size cap exceeded returns 400",
				url: "/v1beta1/projects/" + testProject.Name +
					"/stage-health-outputs?" + manyStageNamesQuery(maxStageHealthOutputsBatch+1),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
		},
	)
}
