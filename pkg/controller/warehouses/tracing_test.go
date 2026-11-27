package warehouses

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

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

func TestSyncWarehouse_tracing(t *testing.T) {
	alreadyExists := apierrors.NewAlreadyExists(
		kargoapi.GroupVersion.WithResource("freights").GroupResource(),
		"fake-freight",
	)
	testCases := []struct {
		name      string
		createErr error
		assert    func(*testing.T, []any, sdktrace.ReadOnlySpan, error)
	}{
		{
			name: "created",
			assert: func(t *testing.T, expected []any, span sdktrace.ReadOnlySpan, err error) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Subset(
					t,
					span.Attributes(),
					append(expected, telemetry.FreightCreatedKey.Bool(true)),
				)
			},
		},
		{
			// The Warehouse handles an existing Freight: not the span's error.
			name:      "already exists",
			createErr: alreadyExists,
			assert: func(t *testing.T, expected []any, span sdktrace.ReadOnlySpan, err error) {
				require.NoError(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
				require.Subset(
					t,
					span.Attributes(),
					append(expected, telemetry.FreightCreatedKey.Bool(false)),
				)
			},
		},
		{
			// A canceled context is not the span's error.
			name:      "canceled",
			createErr: context.Canceled,
			assert: func(t *testing.T, _ []any, span sdktrace.ReadOnlySpan, err error) {
				require.Error(t, err)
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
		{
			name:      "create fails",
			createErr: errors.New("something went wrong"),
			assert: func(t *testing.T, _ []any, span sdktrace.ReadOnlySpan, err error) {
				require.Error(t, err)
				require.Equal(t, codes.Error, span.Status().Code)
				require.Contains(t, span.Attributes(), telemetry.FreightCreatedKey.Bool(false))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			var created *kargoapi.Freight
			r := &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{{
							RepoURL: "fake-repo",
							Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
						}},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{Namespace: "fake-namespace"},
						Commits:    []kargoapi.GitCommit{{ID: "fake-commit"}},
					}, nil
				},
				createFreightFn: func(
					_ context.Context,
					obj client.Object,
					_ ...client.CreateOption,
				) error {
					created = obj.(*kargoapi.Freight) // nolint: forcetypeassert
					return testCase.createErr
				},
				patchStatusFn: func(
					context.Context,
					*kargoapi.Warehouse,
					func(*kargoapi.WarehouseStatus),
				) error {
					return nil
				},
			}
			warehouse := &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "fake-namespace",
					Name:       "fake-warehouse",
					Generation: 1,
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			}
			ctx, parent := otel.Tracer("test").Start(t.Context(), "Reconcile Warehouse")
			_, err := r.syncWarehouse(ctx, warehouse)
			parent.End()

			spans := ended()
			require.Contains(
				t,
				telemetrytest.EndedSpan(t, spans, "Discover artifacts").Attributes(),
				telemetry.SubscriptionsKey.Int(0),
			)
			require.NotNil(t, created)
			// The name the Freight webhook will give it, on the reconcile span too.
			expected := []any{
				telemetry.FreightKey.String(api.GenerateFreightID(created)),
				telemetry.FreightOriginKey.String("Warehouse/fake-warehouse"),
				telemetry.VCSCommitKey.String("fake-commit"),
			}
			require.Subset(
				t,
				telemetrytest.EndedSpan(t, spans, "Reconcile Warehouse").Attributes(),
				expected,
			)
			testCase.assert(
				t,
				expected,
				telemetrytest.EndedSpan(t, spans, "Create Freight"),
				err,
			)
		})
	}
}

func TestSyncWarehouse_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// discovery is off by default.
	telemetrytest.SetCategories(t, "default")
	var created bool
	r := &reconciler{
		discoverArtifactsFn: func(
			context.Context,
			string,
			[]kargoapi.RepoSubscription,
			*kargoapi.DiscoveredArtifacts,
		) (*kargoapi.DiscoveredArtifacts, error) {
			return &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{{
					RepoURL: "fake-repo",
					Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
				}},
			}, nil
		},
		buildFreightFromLatestArtifactsFn: func(
			string,
			*kargoapi.DiscoveredArtifacts,
		) (*kargoapi.Freight, error) {
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: "fake-namespace"},
				Commits:    []kargoapi.GitCommit{{ID: "fake-commit"}},
			}, nil
		},
		createFreightFn: func(context.Context, client.Object, ...client.CreateOption) error {
			created = true
			return nil
		},
		patchStatusFn: func(
			context.Context,
			*kargoapi.Warehouse,
			func(*kargoapi.WarehouseStatus),
		) error {
			return nil
		},
	}
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "fake-namespace",
			Name:       "fake-warehouse",
			Generation: 1,
		},
		Spec: kargoapi.WarehouseSpec{
			FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
		},
	}
	ctx, parent := otel.Tracer("test").Start(t.Context(), "Reconcile Warehouse")
	_, err := r.syncWarehouse(ctx, warehouse)
	require.NoError(t, err)
	require.True(t, created)
	parent.End()

	// Neither Discover artifacts nor Create Freight, and no Freight on the
	// reconcile span.
	spans := ended()
	require.Len(t, spans, 1)
	require.Equal(t, "Reconcile Warehouse", spans[0].Name())
	require.Empty(t, spans[0].Attributes())
}

func TestDiscoverySpan_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// discovery is off by default.
	telemetrytest.SetCategories(t, "default")
	ctx := t.Context()
	sub := kargoapi.RepoSubscription{
		Git: &kargoapi.GitSubscription{RepoURL: "https://user:pass@example.com/repo"},
	}
	got, span := startDiscoverySpan(ctx, sub)
	require.Equal(t, ctx, got)
	endDiscoverySpan(t.Context(), span, kargoapi.GitDiscoveryResult{}, nil)
	require.Empty(t, ended())
	// Nothing is computed for a span that won't be recorded.
	require.Zero(t, testing.AllocsPerRun(10, func() {
		_, s := startDiscoverySpan(ctx, sub)
		s.End()
	}))
}

func TestDiscoverySpan(t *testing.T) {
	testCases := []struct {
		name   string
		sub    kargoapi.RepoSubscription
		res    any
		err    error
		assert func(*testing.T, sdktrace.ReadOnlySpan)
	}{
		{
			name: "git",
			sub: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{RepoURL: "https://user:pass@example.com/repo"},
			},
			res: kargoapi.GitDiscoveryResult{
				Commits: []kargoapi.DiscoveredCommit{{ID: "new"}, {ID: "old"}},
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "Discover git", span.Name())
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.SubscriptionKindKey.String("git"),
						telemetry.SubscriptionRepoKey.String("https://example.com/repo"),
						telemetry.DiscoveredKey.Int(2),
						telemetry.DiscoveredNewestKey.String("new"),
						telemetry.VCSCommitKey.String("new"),
					},
				)
			},
		},
		{
			name: "image",
			sub: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{RepoURL: "ghcr.io/example/app"},
			},
			res: kargoapi.ImageDiscoveryResult{
				References: []kargoapi.DiscoveredImageReference{{Tag: "v2"}},
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "Discover image", span.Name())
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.DiscoveredKey.Int(1),
						telemetry.DiscoveredNewestKey.String("v2"),
					},
				)
			},
		},
		{
			name: "chart",
			sub: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{RepoURL: "oci://ghcr.io/example/chart"},
			},
			res: kargoapi.ChartDiscoveryResult{Versions: []string{"1.2.0", "1.1.0"}},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "Discover chart", span.Name())
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						telemetry.DiscoveredKey.Int(2),
						telemetry.DiscoveredNewestKey.String("1.2.0"),
					},
				)
			},
		},
		{
			name: "other",
			res: kargoapi.DiscoveryResult{
				ArtifactReferences: []kargoapi.ArtifactReference{{}},
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "Discover other", span.Name())
				require.Contains(t, span.Attributes(), telemetry.DiscoveredKey.Int(1))
			},
		},
		{
			name: "nothing discovered",
			sub: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{RepoURL: "https://example.com/repo"},
			},
			res: kargoapi.GitDiscoveryResult{},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Contains(t, span.Attributes(), telemetry.DiscoveredKey.Int(0))
				for _, a := range span.Attributes() {
					require.NotEqual(t, telemetry.DiscoveredNewestKey, a.Key)
				}
			},
		},
		{
			name: "error",
			sub: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{RepoURL: "https://example.com/repo"},
			},
			err: errors.New("something went wrong"),
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
		{
			name: "canceled",
			sub: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{RepoURL: "https://example.com/repo"},
			},
			err: fmt.Errorf("error listing refs: %w", context.Canceled),
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			ctx, span := startDiscoverySpan(t.Context(), testCase.sub)
			endDiscoverySpan(ctx, span, testCase.res, testCase.err)
			spans := ended()
			require.Len(t, spans, 1)
			testCase.assert(t, spans[0])
		})
	}
}

func TestEndDiscoverySpan_notRecording(t *testing.T) {
	t.Parallel()
	// A span that isn't recording is ended without computing attributes.
	span := trace.SpanFromContext(t.Context())
	require.False(t, span.IsRecording())
	endDiscoverySpan(t.Context(), span, kargoapi.GitDiscoveryResult{}, nil)
}
