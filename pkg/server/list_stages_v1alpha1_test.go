package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestListStages(t *testing.T) {
	testNamespace := mustNewObject[corev1.Namespace]("testdata/namespace.yaml")
	// Direct subscriber: gets freight directly from the warehouse
	stageDirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stage-direct-wh1",
			Namespace: "kargo-demo",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	// Indirect subscriber: gets freight from warehouse-1 via an upstream stage
	stageIndirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stage-indirect-wh1",
			Namespace: "kargo-demo",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Stages: []string{"stage-direct-wh1"}},
			}},
		},
	}
	stageWH2 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stage-wh2",
			Namespace: "kargo-demo",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-2",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	stageBothWarehouses := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stage-both",
			Namespace: "kargo-demo",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Sources: kargoapi.FreightSources{Direct: true},
				},
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
					Sources: kargoapi.FreightSources{Direct: true},
				},
			},
		},
	}

	testCases := []struct {
		name         string
		req          *svcv1alpha1.ListStagesRequest
		objects      []client.Object
		interceptor  interceptor.Funcs
		errExpected  bool
		expectedCode connect.Code
		assert       func(*testing.T, *connect.Response[svcv1alpha1.ListStagesResponse])
	}{
		{
			name: "empty project",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "non-existing project",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "non-existing-project",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		{
			name: "lists all stages when no warehouse filter",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				testNamespace,
				stageDirectWH1,
				stageWH2,
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Len(t, res.Msg.GetStages(), 2)
			},
		},
		{
			name: "filters stages by warehouse including indirect subscribers",
			req: &svcv1alpha1.ListStagesRequest{
				Project:        "kargo-demo",
				FreightOrigins: []string{"warehouse-1"},
			},
			objects: []client.Object{
				testNamespace,
				stageDirectWH1,
				stageIndirectWH1,
				stageWH2,
				stageBothWarehouses,
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Len(t, res.Msg.GetStages(), 3)
				names := make([]string, len(res.Msg.GetStages()))
				for i, s := range res.Msg.GetStages() {
					names[i] = s.GetName()
				}
				require.Contains(t, names, "stage-direct-wh1")
				require.Contains(t, names, "stage-indirect-wh1")
				require.Contains(t, names, "stage-both")
			},
		},
		{
			name: "filters stages by multiple warehouses deduplicates",
			req: &svcv1alpha1.ListStagesRequest{
				Project:        "kargo-demo",
				FreightOrigins: []string{"warehouse-1", "warehouse-2"},
			},
			objects: []client.Object{
				testNamespace,
				stageDirectWH1,
				stageWH2,
				stageBothWarehouses,
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				// stageBothWarehouses subscribes to both warehouses but should
				// appear only once.
				require.Len(t, res.Msg.GetStages(), 3)
			},
		},
		{
			name: "returns empty when no stages match warehouse filter",
			req: &svcv1alpha1.ListStagesRequest{
				Project:        "kargo-demo",
				FreightOrigins: []string{"non-existent-warehouse"},
			},
			objects: []client.Object{
				testNamespace,
				stageDirectWH1,
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Empty(t, res.Msg.GetStages())
			},
		},
		{
			name: "summary=true strips heavy fields from each Stage",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "kargo-demo",
				Summary: true,
			},
			objects: []client.Object{
				testNamespace,
				stageWithHeavyFields(),
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Len(t, res.Msg.GetStages(), 1)
				assertStageStripped(t, res.Msg.GetStages()[0])
			},
		},
		{
			name: "summary=false retains all fields",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				testNamespace,
				stageWithHeavyFields(),
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Len(t, res.Msg.GetStages(), 1)
				assertStageNotStripped(t, res.Msg.GetStages()[0])
			},
		},
		{
			name: "ResourceVersion is propagated from underlying StageList",
			req: &svcv1alpha1.ListStagesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				testNamespace,
				stageDirectWH1,
			},
			interceptor: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					if err := c.List(ctx, list, opts...); err != nil {
						return err
					}
					if sl, ok := list.(*kargoapi.StageList); ok {
						sl.ResourceVersion = "fake-rv-77"
					}
					return nil
				},
			},
			assert: func(t *testing.T, res *connect.Response[svcv1alpha1.ListStagesResponse]) {
				require.Equal(t, "fake-rv-77", res.Msg.GetResourceVersion())
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			c, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
						string,
					) (client.WithWatch, error) {
						b := fake.NewClientBuilder().WithScheme(mustNewScheme())
						if len(tc.objects) > 0 {
							copies := make([]client.Object, len(tc.objects))
							for i, obj := range tc.objects {
								objCopy, ok := obj.DeepCopyObject().(client.Object)
								require.True(t, ok)
								copies[i] = objCopy
							}
							b = b.WithObjects(copies...)
						}
						b = b.WithInterceptorFuncs(tc.interceptor)
						return b.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: c,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := svr.ListStages(ctx, connect.NewRequest(tc.req))
			if tc.errExpected {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			tc.assert(t, res)
		})
	}
}

func Test_server_listStages(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	// Direct subscriber: gets freight directly from the warehouse
	stageDirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "stage-direct-wh1",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	// Indirect subscriber: gets freight from warehouse-1 via an upstream stage
	stageIndirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "stage-indirect-wh1",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Stages: []string{"stage-direct-wh1"}},
			}},
		},
	}
	stageWH2 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "stage-wh2",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-2",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
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
				name: "lists all Stages without warehouse filter",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageDirectWH1,
					stageWH2,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					err := json.Unmarshal(w.Body.Bytes(), stages)
					require.NoError(t, err)
					require.Len(t, stages.Items, 2)
				},
			},
			{
				name: "filters Stages by warehouse including indirect subscribers",
				url:  "/v1beta1/projects/" + testProject.Name + "/stages?freightOrigins=warehouse-1",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageDirectWH1,
					stageIndirectWH1,
					stageWH2,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					err := json.Unmarshal(w.Body.Bytes(), stages)
					require.NoError(t, err)
					require.Len(t, stages.Items, 2)
					names := []string{stages.Items[0].Name, stages.Items[1].Name}
					require.Contains(t, names, "stage-direct-wh1")
					require.Contains(t, names, "stage-indirect-wh1")
				},
			},
			{
				name: "filters Stages by multiple warehouses",
				url:  "/v1beta1/projects/" + testProject.Name + "/stages?freightOrigins=warehouse-1&freightOrigins=warehouse-2",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageDirectWH1,
					stageWH2,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					err := json.Unmarshal(w.Body.Bytes(), stages)
					require.NoError(t, err)
					require.Len(t, stages.Items, 2)
				},
			},
			{
				name: "summary=true strips heavy fields",
				url:  "/v1beta1/projects/" + testProject.Name + "/stages?summary=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageWithHeavyFieldsIn(testProject.Name),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), stages))
					require.Len(t, stages.Items, 1)
					assertStageStripped(t, &stages.Items[0])
				},
			},
			{
				name: "summary=false retains heavy fields",
				url:  "/v1beta1/projects/" + testProject.Name + "/stages",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					stageWithHeavyFieldsIn(testProject.Name),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), stages))
					require.Len(t, stages.Items, 1)
					assertStageNotStripped(t, &stages.Items[0])
				},
			},
			{
				name: "ListMeta.ResourceVersion is propagated to REST response body",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, stageDirectWH1).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(
							ctx context.Context,
							c client.WithWatch,
							list client.ObjectList,
							opts ...client.ListOption,
						) error {
							if err := c.List(ctx, list, opts...); err != nil {
								return err
							}
							if sl, ok := list.(*kargoapi.StageList); ok {
								sl.ResourceVersion = "fake-rv-rest"
							}
							return nil
						},
					}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					stages := &kargoapi.StageList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), stages))
					require.Equal(t, "fake-rv-rest", stages.ResourceVersion)
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
				name: "filters watch events by warehouse including indirect subscribers",
				url:  "/v1beta1/projects/" + projectName + "/stages?watch=true&freightOrigins=warehouse-1",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Direct subscriber (should be sent)
					_ = c.Create(ctx, &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-direct-wh1",
						},
						Spec: kargoapi.StageSpec{
							RequestedFreight: []kargoapi.FreightRequest{{
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: "warehouse-1",
								},
								Sources: kargoapi.FreightSources{Direct: true},
							}},
						},
					})
					// Indirect subscriber via upstream stage (should also be sent)
					_ = c.Create(ctx, &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-indirect-wh1",
						},
						Spec: kargoapi.StageSpec{
							RequestedFreight: []kargoapi.FreightRequest{{
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: "warehouse-1",
								},
								Sources: kargoapi.FreightSources{
									Stages: []string{"stage-direct-wh1"},
								},
							}},
						},
					})
					// Stage in a different warehouse (should be filtered out)
					_ = c.Create(ctx, &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "stage-wh2",
						},
						Spec: kargoapi.StageSpec{
							RequestedFreight: []kargoapi.FreightRequest{{
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: "warehouse-2",
								},
								Sources: kargoapi.FreightSources{Direct: true},
							}},
						},
					})
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

					body := w.Body.String()
					require.Contains(t, body, "stage-direct-wh1")
					require.Contains(t, body, "stage-indirect-wh1")
					require.NotContains(t, body, "stage-wh2")
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
			{
				name: "summary=true strips heavy fields from streamed Stages",
				url:  "/v1beta1/projects/" + projectName + "/stages?watch=true&summary=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}},
				),
				operations: func(ctx context.Context, c client.Client) {
					_ = c.Create(ctx, stageWithHeavyFieldsIn(projectName))
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					body := w.Body.String()
					// Heavy field markers must not appear; skeleton markers should.
					require.Contains(t, body, "stage-heavy")
					require.NotContains(t, body, `"output":{"app":"x"}`)
					require.NotContains(t, body, `"config":{"k":"v"}`)
					require.NotContains(t, body, `"if":"true"`)
				},
			},
			{
				name: "summary=false retains heavy fields in streamed Stages",
				url:  "/v1beta1/projects/" + projectName + "/stages?watch=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}},
				),
				operations: func(ctx context.Context, c client.Client) {
					_ = c.Create(ctx, stageWithHeavyFieldsIn(projectName))
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					body := w.Body.String()
					require.Contains(t, body, "stage-heavy")
					require.Contains(t, body, `"output":{"app":"x"}`)
					require.Contains(t, body, `"config":{"k":"v"}`)
				},
			},
			{
				name: "resourceVersion query param is forwarded to underlying watch",
				url:  "/v1beta1/projects/" + projectName + "/stages?watch=true&resourceVersion=fake-rv-123",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}).
					WithInterceptorFuncs(interceptor.Funcs{
						Watch: func(
							ctx context.Context,
							c client.WithWatch,
							obj client.ObjectList,
							opts ...client.ListOption,
						) (watch.Interface, error) {
							seen := ""
							for _, o := range opts {
								if rawOpts, ok := o.(*client.ListOptions); ok && rawOpts.Raw != nil {
									seen = rawOpts.Raw.ResourceVersion
								}
							}
							if seen != "fake-rv-123" {
								return nil, fmt.Errorf(
									"expected resourceVersion=fake-rv-123, got %q",
									seen,
								)
							}
							return c.Watch(ctx, obj, opts...)
						},
					}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
				},
			},
		},
	)
}

func stageWithHeavyFields() *kargoapi.Stage {
	return stageWithHeavyFieldsIn("kargo-demo")
}

func stageWithHeavyFieldsIn(namespace string) *kargoapi.Stage {
	const name = "stage-heavy"
	rawConfig := &apiextensionsv1.JSON{Raw: []byte(`{"k":"v"}`)}
	rawOutput := &apiextensionsv1.JSON{Raw: []byte(`{"app":"x"}`)}
	return &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses:   "copy",
						As:     "step-0",
						Config: rawConfig,
						If:     "true",
						Vars:   []kargoapi.ExpressionVariable{{Name: "v", Value: "1"}},
					}},
				},
			},
		},
		Status: kargoapi.StageStatus{
			FreightHistory: kargoapi.FreightHistory{
				{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f0"}}},
				{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f1"}}},
			},
			Health: &kargoapi.Health{
				Status: kargoapi.HealthStateHealthy,
				Output: rawOutput,
			},
		},
	}
}

func assertStageStripped(t *testing.T, s *kargoapi.Stage) {
	t.Helper()
	require.LessOrEqual(t, len(s.Status.FreightHistory), 1)
	require.NotNil(t, s.Status.Health)
	require.Nil(t, s.Status.Health.Output)
	require.NotNil(t, s.Spec.PromotionTemplate)
	for i, step := range s.Spec.PromotionTemplate.Spec.Steps {
		require.Nilf(t, step.Config, "step %d config", i)
		require.Emptyf(t, step.If, "step %d if", i)
		require.Nilf(t, step.Vars, "step %d vars", i)
	}
}

func assertStageNotStripped(t *testing.T, s *kargoapi.Stage) {
	t.Helper()
	require.NotNil(t, s.Status.Health)
	require.NotNil(t, s.Status.Health.Output)
	require.Greater(t, len(s.Status.FreightHistory), 1)
	require.NotNil(t, s.Spec.PromotionTemplate)
	require.NotEmpty(t, s.Spec.PromotionTemplate.Spec.Steps)
	require.NotNil(t, s.Spec.PromotionTemplate.Spec.Steps[0].Config)
}
