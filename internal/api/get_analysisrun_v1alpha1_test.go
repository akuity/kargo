package api

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

	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetAnalysisRun(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.GetAnalysisRunRequest
		getAnalysisRunFn func(context.Context, client.Client, types.NamespacedName) (*rollouts.AnalysisRun, error)
		errExpected      bool
		expectedCode     connect.Code
	}{
		"empty namespace": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "",
				Name:      "",
			},
			getAnalysisRunFn: rollouts.GetAnalysisRun,
			errExpected:      true,
			expectedCode:     connect.CodeInvalidArgument,
		},
		"empty name": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "",
			},
			getAnalysisRunFn: rollouts.GetAnalysisRun,
			errExpected:      true,
			expectedCode:     connect.CodeInvalidArgument,
		},
		"existing AnalysisRun": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
			},
			getAnalysisRunFn: rollouts.GetAnalysisRun,
		},
		"non-existing namespace": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-x",
				Name:      "test",
			},
			getAnalysisRunFn: rollouts.GetAnalysisRun,
			errExpected:      true,
			expectedCode:     connect.CodeNotFound,
		},
		"non-existing AnalysisRun": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "non-existing",
			},
			getAnalysisRunFn: rollouts.GetAnalysisRun,
			errExpected:      true,
			expectedCode:     connect.CodeNotFound,
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.GetAnalysisRunRequest{
				Namespace: "kargo-demo",
				Name:      "test",
			},
			getAnalysisRunFn: nil,
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
								mustNewObject[rollouts.AnalysisRun]("testdata/analysisrun.yaml"),
							).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:           client,
				getAnalysisRunFn: testCase.getAnalysisRunFn,
			}
			res, err := (svr).GetAnalysisRun(ctx, connect.NewRequest(testCase.req))
			if testCase.errExpected {
				require.Error(t, err)
				require.Equal(t, testCase.expectedCode, connect.CodeOf(err))
				return
			}
			require.NotNil(t, res.Msg.GetAnalysisRun())
			require.Equal(t, testCase.req.GetNamespace(), res.Msg.GetAnalysisRun().Namespace)
			require.Equal(t, testCase.req.GetName(), res.Msg.GetAnalysisRun().Name)
		})
	}
}
