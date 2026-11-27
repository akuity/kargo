package kubeclient

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/telemetry"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestPatchStatus(t *testing.T) {
	testCases := []struct {
		name string
		// status is the Stage's status before the update.
		status kargoapi.StageStatus
		update func(*kargoapi.StageStatus)
		// expectPatch is the status patch sent, or "" when none is.
		expectPatch string
	}{
		{
			name:   "no change: no patch",
			status: kargoapi.StageStatus{FreightSummary: "ok", LastHandledRefresh: "r1"},
			update: func(*kargoapi.StageStatus) {},
		},
		{
			name:   "same value set again: no patch",
			status: kargoapi.StageStatus{FreightSummary: "ok"},
			update: func(s *kargoapi.StageStatus) { s.FreightSummary = "ok" },
		},
		{
			name:   "nil and empty slice are equal: no patch",
			status: kargoapi.StageStatus{},
			update: func(s *kargoapi.StageStatus) { s.Conditions = []metav1.Condition{} },
		},
		{
			name:        "changed field: patched",
			status:      kargoapi.StageStatus{FreightSummary: "old", LastHandledRefresh: "r1"},
			update:      func(s *kargoapi.StageStatus) { s.FreightSummary = "new" },
			expectPatch: `{"status":{"freightSummary":"new"}}`,
		},
		{
			name:        "cleared field: patched",
			status:      kargoapi.StageStatus{FreightSummary: "old"},
			update:      func(s *kargoapi.StageStatus) { s.FreightSummary = "" },
			expectPatch: `{"status":{"freightSummary":null}}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			stage := &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
				Status:     testCase.status,
			}
			var patches []string
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(stage.DeepCopy()).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						ctx context.Context,
						c client.Client,
						sub string,
						obj client.Object,
						patch client.Patch,
						opts ...client.SubResourcePatchOption,
					) error {
						data, err := patch.Data(obj)
						require.NoError(t, err)
						patches = append(patches, string(data))
						return c.SubResource(sub).Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()
			original := stage.Status.DeepCopy()

			require.NoError(t, PatchStatus(t.Context(), c, stage, testCase.update))

			if testCase.expectPatch == "" {
				require.Empty(t, patches)
				// Nothing was sent, so the caller's object is untouched.
				require.Equal(t, original, &stage.Status)
				return
			}
			require.Len(t, patches, 1)
			require.JSONEq(t, testCase.expectPatch, patches[0])
			got := &kargoapi.Stage{}
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(stage), got))
			want := original.DeepCopy()
			testCase.update(want)
			gotJSON, err := json.Marshal(got.Status)
			require.NoError(t, err)
			wantJSON, err := json.Marshal(want)
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}

func TestPatchStatus_tracing(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	newFreight := func() *kargoapi.Freight {
		return &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "wh",
			},
		}
	}
	addStage := func(status *kargoapi.FreightStatus) {
		status.AddCurrentStage("dev", metav1.Now().Time)
	}
	failingClient := func(patchErr error) func(*kargoapi.Freight) client.Client {
		return func(f *kargoapi.Freight) client.Client {
			return fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(f).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						context.Context,
						client.Client,
						string,
						client.Object,
						client.Patch,
						...client.SubResourcePatchOption,
					) error {
						return patchErr
					},
				}).
				Build()
		}
	}
	testCases := []struct {
		name   string
		client func(*kargoapi.Freight) client.Client
		update func(*kargoapi.FreightStatus)
		assert func(*testing.T, sdktrace.ReadOnlySpan, error)
	}{
		{
			name: "patched",
			client: func(f *kargoapi.Freight) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(f).
					WithStatusSubresource(f).
					Build()
			},
			update: addStage,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, err error) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.KindKey.String("Freight"),
						telemetry.StageKey.String("dev"),
						telemetry.FreightKey.String("abc"),
						telemetry.WarehouseKey.String("wh"),
						telemetry.PatchNoopKey.Bool(false),
						telemetry.PatchFieldsKey.StringSlice([]string{"currentlyIn"}),
					},
				)
			},
		},
		{
			name: "no-op",
			client: func(f *kargoapi.Freight) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(f).Build()
			},
			update: func(*kargoapi.FreightStatus) {},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, err error) {
				require.NoError(t, err)
				require.Contains(t, span.Attributes(), telemetry.PatchNoopKey.Bool(true))
			},
		},
		{
			name:   "patch fails",
			client: failingClient(apierrors.NewInternalError(errors.New("boom"))),
			update: addStage,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, err error) {
				require.Error(t, err)
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
		{
			name: "not found",
			client: func(*kargoapi.Freight) client.Client {
				// The Freight doesn't exist.
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			update: addStage,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, err error) {
				require.True(t, apierrors.IsNotFound(err))
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name:   "canceled",
			client: failingClient(context.Canceled),
			update: addStage,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, err error) {
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ended := telemetrytest.RecordSpans(t)
			freight := newFreight()
			ctx := telemetry.ContextWithAttributes(
				context.Background(),
				telemetry.StageKey.String("dev"),
			)
			err := PatchStatus(ctx, testCase.client(freight), freight, testCase.update)
			spans := ended()
			testCase.assert(t, telemetrytest.EndedSpan(t, spans, "PatchStatus"), err)
		})
	}
}

func TestPatchStatus_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	// patch is off by default.
	for _, categories := range []string{"default", "none"} {
		t.Run(categories, func(t *testing.T) {
			ended := telemetrytest.RecordSpans(t)
			telemetrytest.SetCategories(t, categories)
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			freight := &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(freight).
				WithStatusSubresource(freight).
				Build()
			ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")
			err := PatchStatus(ctx, c, freight, func(status *kargoapi.FreightStatus) {
				status.AddCurrentStage("dev", metav1.Now().Time)
			})
			require.NoError(t, err)
			require.Empty(t, ended())
			// Nothing lands on the caller's span instead.
			ro, ok := parent.(sdktrace.ReadOnlySpan)
			require.True(t, ok)
			require.Empty(t, ro.Attributes())
			require.True(t, ro.EndTime().IsZero())
			parent.End()
		})
	}
}
