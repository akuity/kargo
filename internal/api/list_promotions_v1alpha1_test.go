package api

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/api/validation"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListPromotions(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.ListPromotionsRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.ListPromotionsResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-promotion",
						Namespace: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotions(), 1)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by creation time": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "oldest-promotion",
						Namespace:         "kargo-demo",
						CreationTimestamp: metav1.NewTime(metav1.Now().Add(-24 * time.Hour)),
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "new-promotion",
						Namespace:         "kargo-demo",
						CreationTimestamp: metav1.Now(),
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "older-promotion",
						Namespace:         "kargo-demo",
						CreationTimestamp: metav1.NewTime(metav1.Now().Add(-2 * time.Hour)),
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "old-promotion",
						Namespace:         "kargo-demo",
						CreationTimestamp: metav1.NewTime(metav1.Now().Add(-1 * time.Hour)),
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotions(), 4)

				// Check that the analysis templates are ordered by time.
				require.Equal(t, "new-promotion", r.Msg.GetPromotions()[0].GetName())
				require.Equal(t, "old-promotion", r.Msg.GetPromotions()[1].GetName())
				require.Equal(t, "older-promotion", r.Msg.GetPromotions()[2].GetName())
				require.Equal(t, "oldest-promotion", r.Msg.GetPromotions()[3].GetName())
			},
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
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := (svr).ListPromotions(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
