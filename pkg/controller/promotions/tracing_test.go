package promotions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient/testutil"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/telemetry"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestReconcile_tracing(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "x"},
		Spec:       kargoapi.PromotionSpec{Stage: "dev", Freight: "abc"},
	}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "wh",
		},
	}
	errBoom := errors.New("something went wrong")
	testCases := []struct {
		name        string
		objects     []client.Object
		freights    []*kargoapi.Freight
		interceptor interceptor.Funcs
		assert      func(*testing.T, error, sdktrace.ReadOnlySpan)
	}{
		{
			name: "error getting the Promotion",
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
						telemetry.PromotionKey.String("x"),
					},
				)
			},
		},
		{
			name: "Promotion not found",
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			// Failing the first status patch stops the reconcile right after
			// the Freight is found.
			name:     "Promotion's Stage and Freight",
			objects:  []client.Object{promo.DeepCopy()},
			freights: []*kargoapi.Freight{freight.DeepCopy()},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return errBoom
				},
			},
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				// Fails here, not on the attributes, if the reconcile stops
				// before the patch.
				require.ErrorIs(t, err, errBoom)
				require.Equal(t, codes.Error, span.Status().Code)
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.StageKey.String("dev"),
						telemetry.FreightKey.String("abc"),
						telemetry.FreightOriginKey.String("Warehouse/wh"),
						telemetry.WarehouseKey.String("wh"),
					},
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithStatusSubresource(&kargoapi.Promotion{}).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			r := &reconciler{
				kargoClient: c,
				apiReader:   c,
				freightClient: &testutil.MockFreightClient{
					GetFreightFun: func(_ context.Context, _, name string) (*kargoapi.Freight, error) {
						for _, fr := range testCase.freights {
							if fr.Name == name {
								return fr, nil
							}
						}
						return nil, nil
					},
				},
				shardPredicate: controller.ResponsibleFor[kargoapi.Promotion]{
					IsDefaultController: true,
				},
			}
			_, err := r.Reconcile(t.Context(), ctrl.Request{
				NamespacedName: types.NamespacedName{Namespace: "smoke", Name: "x"},
			})
			testCase.assert(t, err, telemetrytest.EndedSpan(t, ended(), "Reconcile Promotion"))
		})
	}
}
