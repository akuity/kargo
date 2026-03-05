package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/stubs/rollouts"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestDeleteAnalysisTemplate(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.DeleteAnalysisTemplateRequest
		rolloutsDisabled bool
		errExpected      bool
		expectedCode     connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"empty name": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing project": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing AnalysisTemplate": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
		},
		"non-existing AnalysisTemplate": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "non-existing-project",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.DeleteAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			rolloutsDisabled: true,
			errExpected:      true,
			expectedCode:     connect.CodeUnimplemented,
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
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[rolloutsapi.AnalysisTemplate]("testdata/analysistemplate.yaml"),
							).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				cfg:                       cfg,
				externalValidateProjectFn: validation.ValidateProject,
			}
			_, err = (svr).DeleteAnalysisTemplate(ctx, connect.NewRequest(testCase.req))
			if testCase.errExpected {
				require.Error(t, err)
				require.Equal(t, testCase.expectedCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)

			at, err := rollouts.GetAnalysisTemplate(ctx, client, types.NamespacedName{
				Namespace: testCase.req.Project,
				Name:      testCase.req.Name,
			})
			require.NoError(t, err)
			require.Nil(t, at)
		})
	}
}

func Test_server_deleteAnalysisTemplate(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testTemplate := &rolloutsapi.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-template",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/analysis-templates/"+testTemplate.Name,
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
				name:          "AnalysisTemplate does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes AnalysisTemplate",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTemplate,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the AnalysisTemplate was deleted from the cluster
					template := &rolloutsapi.AnalysisTemplate{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testTemplate),
						template,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
