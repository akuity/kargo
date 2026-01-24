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
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestListAnalysisTemplates(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListAnalysisTemplatesRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetAnalysisTemplates(), 1)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by name": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("z-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("a-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("m-analysistemplate")
					return obj
				}(),
				func() client.Object {
					obj := mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml")
					obj.SetName("0-analysistemplate")
					return obj
				}(),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetAnalysisTemplates(), 4)

				// Check that the analysis templates are ordered by name.
				require.Equal(t, "0-analysistemplate", r.Msg.GetAnalysisTemplates()[0].GetName())
				require.Equal(t, "a-analysistemplate", r.Msg.GetAnalysisTemplates()[1].GetName())
				require.Equal(t, "m-analysistemplate", r.Msg.GetAnalysisTemplates()[2].GetName())
				require.Equal(t, "z-analysistemplate", r.Msg.GetAnalysisTemplates()[3].GetName())
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
			res, err := (svr).ListAnalysisTemplates(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_listAnalysisTemplates(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/analysis-templates",
		[]restTestCase{
			{
				name:          "Rollouts integration disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no AnalysisTemplates exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &rollouts.AnalysisTemplateList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists AnalysisTemplates",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&rollouts.AnalysisTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "template-1",
						},
					},
					&rollouts.AnalysisTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "template-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the AnalysisTemplates in the response
					templates := &rollouts.AnalysisTemplateList{}
					err := json.Unmarshal(w.Body.Bytes(), templates)
					require.NoError(t, err)
					require.Len(t, templates.Items, 2)
				},
			},
		},
	)
}
