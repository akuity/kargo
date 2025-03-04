package server

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
								mustNewObject[rollouts.ClusterAnalysisTemplate]("testdata/clusteranalysistemplate.yaml"),
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
