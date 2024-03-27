package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/api/validation"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListAnalysisTemplates(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListAnalysisTemplatesRequest
		rolloutsDisabled bool
		errExpected      bool
		expectedCode     connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "non-existing-project",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.ListAnalysisTemplatesRequest{
				Project: "kargo-demo",
			},
			rolloutsDisabled: true,
			errExpected:      true,
			expectedCode:     connect.CodeUnimplemented,
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Simulate an admin user to prevent any authz issues with the authorizing
			// client.
			ctx := user.ContextWithInfo(
				context.Background(),
				user.Info{
					IsAdmin: true,
				},
			)

			cfg := config.ServerConfigFromEnv()
			if testCase.rolloutsDisabled {
				cfg.RolloutsIntegrationEnabled = false
			}

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						if err := rollouts.AddToScheme(scheme); err != nil {
							return nil, err
						}

						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
							).
							WithLists(&rollouts.AnalysisTemplateList{
								Items: []rollouts.AnalysisTemplate{
									*mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml"),
								},
							}).
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
			res, err := (svr).ListAnalysisTemplates(ctx, connect.NewRequest(testCase.req))
			if testCase.errExpected {
				require.Error(t, err)
				require.Equal(t, testCase.expectedCode, connect.CodeOf(err))
				return
			}
			require.Len(t, res.Msg.GetAnalysisTemplates(), 1)
		})
	}
}
