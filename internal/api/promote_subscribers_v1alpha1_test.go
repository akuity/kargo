package api

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
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

func TestPromoteSubscribers(t *testing.T) {
	testSets := map[string]struct {
		req                *svcv1alpha1.PromoteSubscribersRequest
		errMsg             string
		expectedCode       connect.Code
		expectedPromotions int32
	}{
		"empty freight": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "upstream",
			},
			errMsg:       "freight should not be empty",
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing Stage": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "does-not-exist",
				Freight: "c353927ca7af42b38c0cdcfa393b2c552740e547",
			},
			errMsg:       `stage "kargo-demo" not found`,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage with non-existing freight": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "upstream",
				Freight: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			errMsg:       `freight "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" not found in Stage`,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage with no subscribers": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "downstream1",
				Freight: "f08b2e72c9b2b7b263da6d55f9536e49b5ce972c",
			},
			errMsg:       `Stage "downstream1" has no subscribers`,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage with unhealthy freight": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "upstream",
				Freight: "abc1237ca7af42b38c0cdcfa393b2c552740e547",
			},
			errMsg:       "Cannot promote freight with health status: Unhealthy",
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing Stage with subscribers": {
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "kargo-demo",
				Stage:   "upstream",
				Freight: "c353927ca7af42b38c0cdcfa393b2c552740e547",
			},
			expectedPromotions: 2,
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
									*mustNewObject[kubev1alpha1.Stage]("testdata/promote-subscribers-upstream.yaml"),
									*mustNewObject[kubev1alpha1.Stage]("testdata/promote-subscribers-downstream1.yaml"),
									*mustNewObject[kubev1alpha1.Stage]("testdata/promote-subscribers-downstream2.yaml"),
								},
							}).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			res, err := (&server{
				client: client,
			}).PromoteSubscribers(ctx, connect.NewRequest(ts.req))
			if ts.errMsg != "" {
				require.ErrorContains(t, err, ts.errMsg)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			assert.Len(t, res.Msg.GetPromotions(), int(ts.expectedPromotions))
			for _, p := range res.Msg.GetPromotions() {
				parts := strings.Split(p.GetMetadata().GetName(), ".")
				require.True(t, strings.HasPrefix(parts[0], "downstream"))
				require.True(t, strings.HasPrefix(p.GetSpec().GetStage(), "downstream"))
				require.Equal(t, parts[2], ts.req.Freight[0:7])
				require.Equal(t, ts.req.GetFreight(), p.GetSpec().GetState())

				var actual kubev1alpha1.Promotion
				require.NoError(t, client.Get(ctx, libClient.ObjectKey{
					Namespace: ts.req.GetProject(),
					Name:      p.GetMetadata().GetName(),
				}, &actual))
			}

		})
	}
}
