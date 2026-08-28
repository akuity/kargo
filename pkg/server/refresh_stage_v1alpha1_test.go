package server

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestRefreshStage(t *testing.T) {
	// baseObjects returns fresh fixture objects on every call -- these tests
	// run in parallel, and fake.NewClientBuilder mutates the objects it's
	// given, so sharing one set of object pointers across subtests races.
	baseObjects := func() []client.Object {
		return []client.Object{
			mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			mustNewObject[kargoapi.Stage]("testdata/stage.yaml"),
		}
	}
	testSets := map[string]struct {
		req         *svcv1alpha1.RefreshStageRequest
		objects     []client.Object
		authorizeFn func(
			context.Context,
			string,
			schema.GroupVersionResource,
			string,
			client.ObjectKey,
		) error
		errExpected   bool
		expectedCode  connect.Code
		wantForbidden bool
		// checkPartialEffect, when set, is called after the RPC call with the
		// underlying internal client, to verify what did or didn't get
		// refreshed -- used by the current-Promotion-cascade cases.
		checkPartialEffect func(*testing.T, client.Client)
	}{
		"empty project": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "",
				Name:    "",
			},
			objects:      baseObjects(),
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"empty name": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			objects:      baseObjects(),
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing project": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			objects:      baseObjects(),
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"non-existing Stage": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "non-existing-project",
				Name:    "test",
			},
			objects:      baseObjects(),
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing Stage": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: baseObjects(),
		},
		"not authorized": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects:       baseObjects(),
			authorizeFn:   refreshDeniedFn(t, "stages"),
			errExpected:   true,
			wantForbidden: true,
		},
		"current promotion refresh not authorized": {
			req: &svcv1alpha1.RefreshStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Namespace: "kargo-demo", Name: "test"},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{Name: "promo-1"},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{Namespace: "kargo-demo", Name: "promo-1"},
				},
			},
			authorizeFn: func(
				_ context.Context,
				_ string,
				gvr schema.GroupVersionResource,
				_ string,
				_ client.ObjectKey,
			) error {
				if gvr.Resource == "promotions" {
					return apierrors.NewForbidden(
						kargoapi.GroupVersion.WithResource("promotions").GroupResource(),
						"promo-1",
						nil,
					)
				}
				require.Equal(t, "stages", gvr.Resource)
				return nil
			},
			errExpected:   true,
			wantForbidden: true,
			checkPartialEffect: func(t *testing.T, internalClient client.Client) {
				// The Stage refresh was already authorized and performed.
				var stage kargoapi.Stage
				require.NoError(t, internalClient.Get(
					t.Context(), client.ObjectKey{Namespace: "kargo-demo", Name: "test"}, &stage,
				))
				require.NotEmpty(t, stage.Annotations[kargoapi.AnnotationKeyRefresh])

				// The current Promotion was NOT refreshed.
				var promo kargoapi.Promotion
				require.NoError(t, internalClient.Get(
					t.Context(), client.ObjectKey{Namespace: "kargo-demo", Name: "promo-1"}, &promo,
				))
				require.Empty(t, promo.Annotations[kargoapi.AnnotationKeyRefresh])
			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			var internalClient client.Client
			kubeClient, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						internalClient = fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(ts.objects...).
							Build()
						return internalClient, nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: kubeClient,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			svr.authorizeFn = kubeClient.Authorize
			if ts.authorizeFn != nil {
				svr.authorizeFn = ts.authorizeFn
			}
			res, err := svr.RefreshStage(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				if ts.wantForbidden {
					require.True(t, apierrors.IsForbidden(err), "expected a Forbidden error, got: %v", err)
				} else {
					require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				}
				if ts.checkPartialEffect != nil {
					ts.checkPartialEffect(t, internalClient)
				}
				return
			}
			require.NoError(t, err)
			stage := res.Msg.GetStage()
			annotation := stage.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
			refreshTime, err := time.Parse(time.RFC3339, annotation)
			require.NoError(t, err)
			// Make sure we set timestamp is close to now
			// Assume it doesn't take 3 seconds to run this unit test.
			require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
			require.Equal(t, ts.req.GetProject(), stage.Namespace)
			require.Equal(t, ts.req.GetName(), stage.Name)
		})
	}
}
