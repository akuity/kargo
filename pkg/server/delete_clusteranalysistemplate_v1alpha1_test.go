package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	rollouts "github.com/akuity/kargo/pkg/api/stubs/rollouts"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestDeleteClusterAnalysisTemplate(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.DeleteClusterAnalysisTemplateRequest
		rolloutsDisabled bool
		errExpected      bool
		expectedCode     connect.Code
	}{
		"empty name": {
			req: &svcv1alpha1.DeleteClusterAnalysisTemplateRequest{
				Name: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing ClusterAnalysisTemplate": {
			req: &svcv1alpha1.DeleteClusterAnalysisTemplateRequest{
				Name: "test",
			},
		},
		"non-existing ClusterAnalysisTemplate": {
			req: &svcv1alpha1.DeleteClusterAnalysisTemplateRequest{
				Name: "non-existing",
			},
			errExpected:  true,
			expectedCode: connect.CodeUnknown,
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.DeleteClusterAnalysisTemplateRequest{
				Name: "test",
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
								mustNewObject[rolloutsapi.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml"),
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
			_, err = (svr).DeleteClusterAnalysisTemplate(ctx, connect.NewRequest(testCase.req))
			if testCase.errExpected {
				require.Error(t, err)
				fmt.Printf("actual: %s, expected: %s", connect.CodeOf(err), testCase.expectedCode)
				require.Equal(t, testCase.expectedCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)

			at, err := rollouts.GetClusterAnalysisTemplate(ctx, client, testCase.req.Name)
			require.NoError(t, err)
			require.Nil(t, at)
		})
	}
}

func Test_server_deleteClusterAnalysisTemplate(t *testing.T) {
	testTemplate := &rolloutsapi.ClusterAnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-template"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodDelete, "/v1beta1/shared/cluster-analysis-templates/"+testTemplate.Name,
		[]restTestCase{
			{
				name:         "Rollouts integration disabled",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "ClusterAnalysisTemplate does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ClusterAnalysisTemplate",
				clientBuilder: fake.NewClientBuilder().WithObjects(testTemplate),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ClusterAnalysisTemplate was deleted from the cluster
					template := &rolloutsapi.ClusterAnalysisTemplate{}
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
