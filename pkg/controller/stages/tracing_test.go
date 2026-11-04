package stages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/telemetry"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestReconcile_tracing(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	errBoom := errors.New("something went wrong")
	testCases := []struct {
		name        string
		interceptor interceptor.Funcs
		assert      func(*testing.T, error, sdktrace.ReadOnlySpan)
	}{
		{
			name: "error getting the Stage",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errBoom
				},
			},
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				require.ErrorIs(t, err, errBoom)
				require.Equal(t, codes.Error, span.Status().Code)
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.ProjectKey.String("smoke"),
						telemetry.StageKey.String("x"),
					},
				)
			},
		},
		{
			name: "Stage not found",
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
	}
	reconcilers := []struct {
		name string
		new  func(client.Client) reconcile.Reconciler
	}{
		{
			name: "regular",
			new: func(c client.Client) reconcile.Reconciler {
				return &RegularStageReconciler{client: c}
			},
		},
		{
			name: "control flow",
			new: func(c client.Client) reconcile.Reconciler {
				return &ControlFlowStageReconciler{client: c}
			},
		},
	}
	for _, rc := range reconcilers {
		for _, testCase := range testCases {
			t.Run(rc.name+"/"+testCase.name, func(t *testing.T) {
				// Not parallel: installs a global tracer provider.
				ended := telemetrytest.RecordSpans(t)
				c := fake.NewClientBuilder().
					WithScheme(scheme).
					WithInterceptorFuncs(testCase.interceptor).
					Build()
				_, err := rc.new(c).Reconcile(t.Context(), ctrl.Request{
					NamespacedName: types.NamespacedName{Namespace: "smoke", Name: "x"},
				})
				testCase.assert(t, err, telemetrytest.EndedSpan(t, ended(), "Reconcile Stage"))
			})
		}
	}
}

func TestRegularStageReconciler_reconcile_tracing(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
	}
	testCases := []struct {
		name    string
		listErr error
		assert  func(*testing.T, []sdktrace.ReadOnlySpan)
	}{
		{
			// The first sub-reconciler fails, in its own span under the caller's.
			name:    "sub-reconciler fails",
			listErr: errors.New("something went wrong"),
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				span := telemetrytest.EndedSpan(t, spans, "Stage: syncing Promotions")
				require.Equal(t, codes.Error, span.Status().Code)
				require.Equal(
					t,
					telemetrytest.EndedSpan(t, spans, "parent").SpanContext().SpanID(),
					span.Parent().SpanID(),
				)
			},
		},
		{
			// A canceled context is not the span's error.
			name:    "canceled",
			listErr: context.Canceled,
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				span := telemetrytest.EndedSpan(t, spans, "Stage: syncing Promotions")
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			r := &RegularStageReconciler{
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(stage).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(
							context.Context,
							client.WithWatch,
							client.ObjectList,
							...client.ListOption,
						) error {
							return testCase.listErr
						},
					}).
					Build(),
			}
			ctx, parent := otel.Tracer("test").Start(t.Context(), "parent")
			_, _, err := r.reconcile(ctx, stage, time.Now())
			require.ErrorIs(t, err, testCase.listErr)
			parent.End()
			testCase.assert(t, ended())
		})
	}
}

func TestCreateAutoPromotion_tracing(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "wh",
		},
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
	}
	testCases := []struct {
		name   string
		client client.Client
		assert func(*testing.T, *kargoapi.Promotion, sdktrace.ReadOnlySpan, error)
	}{
		{
			name:   "created",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assert: func(
				t *testing.T,
				promo *kargoapi.Promotion,
				span sdktrace.ReadOnlySpan,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.FreightKey.String("abc"),
						telemetry.WarehouseKey.String("wh"),
						telemetry.PromotionKey.String(promo.Name),
					},
				)
			},
		},
		{
			// autoPromoteFreight handles a webhook denial: not the span's error.
			name: "denied by an admission webhook",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.CreateOption,
					) error {
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("promotions").GroupResource(),
							"",
							errors.New("denied"),
						)
					},
				}).
				Build(),
			assert: func(
				t *testing.T,
				_ *kargoapi.Promotion,
				span sdktrace.ReadOnlySpan,
				err error,
			) {
				require.True(t, apierrors.IsForbidden(err))
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
		{
			// A canceled context is not the span's error.
			name: "canceled",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.CreateOption,
					) error {
						return context.Canceled
					},
				}).
				Build(),
			assert: func(
				t *testing.T,
				_ *kargoapi.Promotion,
				span sdktrace.ReadOnlySpan,
				err error,
			) {
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
		{
			name: "create fails",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.CreateOption,
					) error {
						return errors.New("something went wrong")
					},
				}).
				Build(),
			assert: func(
				t *testing.T,
				_ *kargoapi.Promotion,
				span sdktrace.ReadOnlySpan,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, codes.Error, span.Status().Code)
				for _, a := range span.Attributes() {
					require.NotEqual(t, telemetry.PromotionKey, a.Key)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			r := &RegularStageReconciler{client: testCase.client}
			promo := api.NewMinimalPromotion(stage, freight.Name)
			err := r.createAutoPromotion(t.Context(), promo, freight)
			span := telemetrytest.EndedSpan(t, ended(), "Create Promotion")
			testCase.assert(t, promo, span, err)
		})
	}
}

func TestRegularStageReconciler_reconcile_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// stage is off by default.
	telemetrytest.SetCategories(t, "default")
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
	}
	r := &RegularStageReconciler{
		client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(stage).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			}).
			Build(),
	}
	ctx, parent := otel.Tracer("test").Start(t.Context(), "parent")
	_, _, err := r.reconcile(ctx, stage, time.Now())
	require.Error(t, err)
	parent.End()

	// No sub-reconciler span, and its error isn't put on the caller's.
	spans := ended()
	require.Len(t, spans, 1)
	require.Equal(t, "parent", spans[0].Name())
	require.Equal(t, codes.Unset, spans[0].Status().Code)
}

func TestStartSubReconcilerSpan_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// stage is off by default.
	telemetrytest.SetCategories(t, "default")
	ctx := t.Context()
	got, span := startSubReconcilerSpan(ctx, "syncing Freight")
	require.Equal(t, ctx, got)
	span.End()
	require.Empty(t, ended())
	// Nothing is computed for a span that won't be recorded.
	require.Zero(t, testing.AllocsPerRun(10, func() {
		_, s := startSubReconcilerSpan(ctx, "syncing Freight")
		s.End()
	}))
}

func TestCreateAutoPromotion_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// stage is off by default.
	telemetrytest.SetCategories(t, "default")
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
	}
	r := &RegularStageReconciler{client: fake.NewClientBuilder().WithScheme(scheme).Build()}
	promo := api.NewMinimalPromotion(stage, "abc")
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
	}
	ctx, parent := otel.Tracer("test").Start(t.Context(), "parent")
	require.NoError(t, r.createAutoPromotion(ctx, promo, freight))
	parent.End()
	spans := ended()
	require.Len(t, spans, 1)
	require.Equal(t, "parent", spans[0].Name())
	require.Empty(t, spans[0].Attributes())
}
