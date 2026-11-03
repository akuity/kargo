package metrics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

func Test_countPromotionsByProject(t *testing.T) {
	t.Parallel()

	promotion := func(project string) kargoapi.Promotion {
		return kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{Namespace: project}}
	}

	testCases := []struct {
		name       string
		promotions []kargoapi.Promotion
		expected   map[string]float64
	}{
		{
			name:     "no Promotions yields no Projects",
			expected: map[string]float64{},
		},
		{
			name: "Promotions are counted per Project",
			promotions: []kargoapi.Promotion{
				promotion("project-a"),
				promotion("project-a"),
				promotion("project-b"),
			},
			expected: map[string]float64{"project-a": 2, "project-b": 1},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, countPromotionsByProject(testCase.promotions))
		})
	}
}

func Test_summarizePromotions(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	promotion := func(project string, phase kargoapi.PromotionPhase, age time.Duration) kargoapi.Promotion {
		return kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         project,
				CreationTimestamp: metav1.Time{Time: t0.Add(-age)},
			},
			Status: kargoapi.PromotionStatus{Phase: phase},
		}
	}

	testCases := []struct {
		name       string
		promotions []kargoapi.Promotion
		expected   promotionSummary
	}{
		{
			name: "no Promotions yields no Projects",
			expected: promotionSummary{
				byPhase:       map[string]map[string]float64{},
				oldestPending: map[string]time.Time{},
			},
		},
		{
			name: "Promotions are counted by phase and the oldest waiting is found",
			promotions: []kargoapi.Promotion{
				promotion("project-a", kargoapi.PromotionPhaseRunning, 30*time.Minute),
				promotion("project-a", kargoapi.PromotionPhasePending, time.Minute),
				promotion("project-a", "", 2*time.Minute),
				promotion("project-a", kargoapi.PromotionPhasePending, 0),
				promotion("project-b", kargoapi.PromotionPhaseRunning, time.Hour),
			},
			expected: promotionSummary{
				byPhase: map[string]map[string]float64{
					"project-a": {"Running": 1, "Pending": 2, "Unknown": 1},
					"project-b": {"Running": 1},
				},
				// Unreconciled counts as pending; Running doesn't.
				oldestPending: map[string]time.Time{"project-a": t0.Add(-2 * time.Minute)},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, summarizePromotions(testCase.promotions))
		})
	}
}

func Test_promotionPhaseCollector(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	promotion := func(name string, phase kargoapi.PromotionPhase, age time.Duration) client.Object {
		return &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "project-a",
				Name:              name,
				CreationTimestamp: metav1.Time{Time: t0.Add(-age)},
			},
			Status: kargoapi.PromotionStatus{Phase: phase},
		}
	}
	var listed []kargoapi.Promotion
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(
			&kargoapi.Promotion{},
			indexer.PromotionsByNonTerminalField,
			indexer.PromotionsByNonTerminal,
		).
		WithObjects(
			promotion("pending", kargoapi.PromotionPhasePending, time.Minute),
			promotion("unknown", "", 2*time.Minute),
			promotion("running", kargoapi.PromotionPhaseRunning, time.Hour),
			promotion("succeeded", kargoapi.PromotionPhaseSucceeded, 3*time.Hour),
			promotion("failed", kargoapi.PromotionPhaseFailed, 3*time.Hour),
			promotion("errored", kargoapi.PromotionPhaseErrored, 3*time.Hour),
			promotion("aborted", kargoapi.PromotionPhaseAborted, 3*time.Hour),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(
				ctx context.Context,
				c client.WithWatch,
				list client.ObjectList,
				opts ...client.ListOption,
			) error {
				if err := c.List(ctx, list, opts...); err != nil {
					return err
				}
				listed = append(listed, list.(*kargoapi.PromotionList).Items...) // nolint: forcetypeassert
				return nil
			},
		}).
		Build()

	// nolint: lll
	const expected = `
# HELP kargo_promotions_by_phase Number of non-terminal Promotions in each phase (Pending, Running or Unknown). A Promotion not yet reconciled is reported in the Unknown phase.
# TYPE kargo_promotions_by_phase gauge
kargo_promotions_by_phase{phase="Pending",project="project-a"} 1
kargo_promotions_by_phase{phase="Running",project="project-a"} 1
kargo_promotions_by_phase{phase="Unknown",project="project-a"} 1
# HELP kargo_promotion_oldest_pending_timestamp_seconds Creation time, as a Unix timestamp, of the oldest Promotion that has not started running (Pending or Unknown phase).
# TYPE kargo_promotion_oldest_pending_timestamp_seconds gauge
kargo_promotion_oldest_pending_timestamp_seconds{project="project-a"} 1.76722548e+09
`
	require.NoError(t, testutil.CollectAndCompare(
		newPromotionPhaseCollector(c), strings.NewReader(expected),
	))
	// Listed through the non-terminal index: terminal Promotions are never read.
	require.NotEmpty(t, listed)
	for _, promo := range listed {
		require.False(t, promo.Status.Phase.IsTerminal(), promo.Name)
	}
}
