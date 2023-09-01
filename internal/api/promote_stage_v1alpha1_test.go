package api

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestPromoteStage(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.PromoteStageRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty state": {
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing Stage": {
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "kargo-demo",
				Name:    "testx",
				State:   "73024971ee9c6daac0ad78aea87803bf332cfdb7",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage with non-existing state": {
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "kargo-demo",
				Name:    "test",
				State:   "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage": {
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "kargo-demo",
				Name:    "test",
				State:   "73024971ee9c6daac0ad78aea87803bf332cfdb7",
			},
		},
	}
	for name, ts := range testSets {
		ts := ts
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
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (libClient.Client, error) {
						return fake.NewClientBuilder().
							WithScheme(mustNewScheme()).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
							).
							WithLists(&kubev1alpha1.StageList{
								Items: []kubev1alpha1.Stage{
									*mustNewObject[kubev1alpha1.Stage]("testdata/stage.yaml"),
								},
							}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			res, err := (&server{
				client: client,
			}).PromoteStage(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}

			require.True(t, strings.HasPrefix(res.Msg.GetPromotion().GetMetadata().GetName(), ts.req.GetName()+"."))
			require.Equal(t, ts.req.GetName(), res.Msg.GetPromotion().GetSpec().GetStage())
			require.Equal(t, ts.req.GetState(), res.Msg.GetPromotion().GetSpec().GetState())

			var actual kubev1alpha1.Promotion
			require.NoError(t, client.Get(ctx, libClient.ObjectKey{
				Namespace: ts.req.GetProject(),
				Name:      res.Msg.GetPromotion().GetMetadata().GetName(),
			}, &actual))
		})
	}
}
