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

func TestListStageSummaries(t *testing.T) {
	const projectName = "kargo-demo"

	newStage := func(name, warehouseName string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: name},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: warehouseName},
					Sources: kargoapi.FreightSources{Direct: true},
				}},
			},
		}
	}

	testCases := map[string]struct {
		req    *svcv1alpha1.ListStageSummariesRequest
		assert func(*testing.T, *connect.Response[svcv1alpha1.ListStageSummariesResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListStageSummariesRequest{Project: ""},
			assert: func(t *testing.T, _ *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListStageSummariesRequest{Project: "does-not-exist"},
			assert: func(t *testing.T, _ *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			},
		},
		"no filter returns all stages sorted by name": {
			req: &svcv1alpha1.ListStageSummariesRequest{Project: projectName},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.NoError(t, err)
				summaries := res.Msg.GetStageSummaries()
				require.Len(t, summaries, 3)
				require.Equal(t, "a-dev", summaries[0].Metadata.Name)
				require.Equal(t, "b-qa", summaries[1].Metadata.Name)
				require.Equal(t, "c-prod", summaries[2].Metadata.Name)
			},
		},
		"single warehouse filter": {
			req: &svcv1alpha1.ListStageSummariesRequest{
				Project:        projectName,
				FreightOrigins: []string{"wh-a"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.NoError(t, err)
				summaries := res.Msg.GetStageSummaries()
				require.Len(t, summaries, 2)
				names := []string{summaries[0].Metadata.Name, summaries[1].Metadata.Name}
				require.ElementsMatch(t, []string{"a-dev", "b-qa"}, names)
			},
		},
		"multiple warehouse filter": {
			req: &svcv1alpha1.ListStageSummariesRequest{
				Project:        projectName,
				FreightOrigins: []string{"wh-a", "wh-b"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.NoError(t, err)
				require.Len(t, res.Msg.GetStageSummaries(), 3)
			},
		},
		"unknown warehouse returns empty": {
			req: &svcv1alpha1.ListStageSummariesRequest{
				Project:        projectName,
				FreightOrigins: []string{"wh-nonexistent"},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, res.Msg.GetStageSummaries())
			},
		},
		"empty filter strings are ignored": {
			req: &svcv1alpha1.ListStageSummariesRequest{
				Project:        projectName,
				FreightOrigins: []string{"", ""},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStageSummariesResponse], err error) {
				require.NoError(t, err)
				// all empty → treated as no filter → return all
				require.Len(t, res.Msg.GetStageSummaries(), 3)
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
								*newStage("a-dev", "wh-a"),
								*newStage("b-qa", "wh-a"),
								*newStage("c-prod", "wh-b"),
							}}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{client: kubeClient}
			svr.externalValidateProjectFn = validation.ValidateProject

			res, err := svr.ListStageSummaries(ctx, connect.NewRequest(tc.req))
			tc.assert(t, res, err)
		})
	}
}

func Test_server_listStageSummaries(t *testing.T) {
	testProject := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}}

	newStage := func(name, warehouseName string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: name},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: warehouseName},
					Sources: kargoapi.FreightSources{Direct: true},
				}},
			},
		}
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/stage-summaries",
		[]restTestCase{
			{
				name: "project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no stages exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					resp := &svcv1alpha1.ListStageSummariesResponse{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
					require.Empty(t, resp.GetStageSummaries())
				},
			},
			{
				name: "lists stage summaries, sorted by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newStage("b-qa", "wh-a"),
					newStage("a-dev", "wh-a"),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					resp := &svcv1alpha1.ListStageSummariesResponse{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
					require.Len(t, resp.GetStageSummaries(), 2)
					require.Equal(t, "a-dev", resp.GetStageSummaries()[0].Metadata.Name)
					require.Equal(t, "b-qa", resp.GetStageSummaries()[1].Metadata.Name)
				},
			},
			{
				name: "applies freightOrigins filter",
				url:  "/v1beta1/projects/" + testProject.Name + "/stage-summaries?freightOrigins=wh-a",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newStage("a-dev", "wh-a"),
					newStage("c-prod", "wh-b"),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					resp := &svcv1alpha1.ListStageSummariesResponse{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
					require.Len(t, resp.GetStageSummaries(), 1)
					require.Equal(t, "a-dev", resp.GetStageSummaries()[0].Metadata.Name)
				},
			},
		},
	)
}

func Test_server_listStageSummaries_watch(t *testing.T) {
	const projectName = "fake-project"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/stage-summaries?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/stage-summaries?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "streams events as SSE",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}},
					&kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: "a"}},
				),
				operations: func(ctx context.Context, c client.Client) {
					_ = c.Create(ctx, &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: "b"},
					})
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Contains(t, w.Body.String(), "data:")
				},
			},
		},
	)
}
