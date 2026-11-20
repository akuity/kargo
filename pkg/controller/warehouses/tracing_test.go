package warehouses

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
			name: "error getting the Warehouse",
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
						telemetry.WarehouseKey.String("x"),
					},
				)
			},
		},
		{
			// A canceled context is not the span's error.
			name: "canceled",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return fmt.Errorf("error getting Warehouse: %w", context.Canceled)
				},
			},
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
		{
			name: "Warehouse not found",
			assert: func(t *testing.T, err error, span sdktrace.ReadOnlySpan) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			r := &reconciler{client: c}
			_, err := r.Reconcile(t.Context(), ctrl.Request{
				NamespacedName: types.NamespacedName{Namespace: "smoke", Name: "x"},
			})
			testCase.assert(t, err, telemetrytest.EndedSpan(t, ended(), "Reconcile Warehouse"))
		})
	}
}
