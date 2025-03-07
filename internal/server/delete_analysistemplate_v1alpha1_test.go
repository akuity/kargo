package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	"github.com/akuity/kargo/internal/api/stubs/rollouts"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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

			ctx := context.Background()

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
					) (client.Client, error) {
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
