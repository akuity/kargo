package server

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestListPromotions(t *testing.T) {
	// We need some promotion names with ULIDs to test the sorting.
	oldestPromotionName := fmt.Sprintf("some-stage.%s.oldest", ulid.Make())
	olderPromotionName := fmt.Sprintf("some-stage.%s.older", ulid.Make())
	oldPromotionName := fmt.Sprintf("some-stage.%s.old", ulid.Make())
	newPromotionName := fmt.Sprintf("some-stage.%s.new", ulid.Make())

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
		"orders by ULID and phase": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oldestPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseRunning,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      olderPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oldPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotions(), 4)

				// Check that the analysis templates are ordered by ULID and phase.
				require.Equal(t, oldestPromotionName, r.Msg.GetPromotions()[0].GetName())
				require.Equal(t, oldPromotionName, r.Msg.GetPromotions()[1].GetName())
				require.Equal(t, newPromotionName, r.Msg.GetPromotions()[2].GetName())
				require.Equal(t, olderPromotionName, r.Msg.GetPromotions()[3].GetName())
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

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
