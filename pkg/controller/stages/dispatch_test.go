package stages

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/dispatch"
)

func TestRegularStageReconciler_resolveCurrentFreight(t *testing.T) {
	t.Parallel()

	nginxDiscovered := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	redisCreated := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	// Two current Freight objects for two origins; the redis one has no
	// DiscoveredAt, so its EffectiveDiscoveredAt falls back to creationTime.
	nginxFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "freight-nginx",
			Namespace: "demo",
		},
		DiscoveredAt: &metav1.Time{Time: nginxDiscovered},
	}
	redisFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "freight-redis",
			Namespace:         "demo",
			CreationTimestamp: metav1.NewTime(redisCreated),
		},
	}

	collection := func(refs map[string]kargoapi.FreightReference) kargoapi.FreightHistory {
		return kargoapi.FreightHistory{{Freight: refs}}
	}

	testCases := []struct {
		name    string
		stage   *kargoapi.Stage
		objects []client.Object
		assert  func(*testing.T, map[string]dispatch.CurrentFreight, error)
	}{
		{
			name: "resolves each origin's current Freight with discovery time",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"},
				Status: kargoapi.StageStatus{
					FreightHistory: collection(map[string]kargoapi.FreightReference{
						"Warehouse/demo/nginx": {Name: "freight-nginx"},
						"Warehouse/demo/redis": {Name: "freight-redis"},
					}),
				},
			},
			objects: []client.Object{nginxFreight, redisFreight},
			assert: func(t *testing.T, got map[string]dispatch.CurrentFreight, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
				assertCurrentFreight(t, got, "Warehouse/demo/nginx", "freight-nginx", nginxDiscovered)
				assertCurrentFreight(t, got, "Warehouse/demo/redis", "freight-redis", redisCreated)
			},
		},
		{
			name: "omits an origin whose current Freight no longer exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"},
				Status: kargoapi.StageStatus{
					FreightHistory: collection(map[string]kargoapi.FreightReference{
						"Warehouse/demo/nginx": {Name: "freight-nginx"},
						"Warehouse/demo/redis": {Name: "freight-gone"},
					}),
				},
			},
			objects: []client.Object{nginxFreight},
			assert: func(t *testing.T, got map[string]dispatch.CurrentFreight, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				assertCurrentFreight(t, got, "Warehouse/demo/nginx", "freight-nginx", nginxDiscovered)
			},
		},
		{
			name: "nil FreightHistory resolves to an empty map",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"},
			},
			assert: func(t *testing.T, got map[string]dispatch.CurrentFreight, err error) {
				require.NoError(t, err)
				require.Empty(t, got)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				Build()
			r := &RegularStageReconciler{client: c}

			got, err := r.resolveCurrentFreight(t.Context(), testCase.stage)
			testCase.assert(t, got, err)
		})
	}
}

// assertCurrentFreight checks a resolved entry, comparing discovery times by
// instant (time.Time.Equal) since a round-trip through the client can change
// the time.Location without changing the instant.
func assertCurrentFreight(
	t *testing.T,
	got map[string]dispatch.CurrentFreight,
	origin, name string,
	discoveredAt time.Time,
) {
	t.Helper()
	cf, ok := got[origin]
	require.True(t, ok, "origin %q missing", origin)
	require.Equal(t, name, cf.Name)
	require.True(t, discoveredAt.Equal(cf.DiscoveredAt),
		"discoveredAt: want %s, got %s", discoveredAt, cf.DiscoveredAt)
}

func TestDispatchEventAnnotations(t *testing.T) {
	t.Parallel()

	until := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 7, 15, 18, 0, 0, 0, time.UTC)

	testCases := []struct {
		name    string
		reasons []dispatch.Reason
		assert  func(*testing.T, map[string]string)
	}{
		{
			name:    "no reasons yields nil",
			reasons: nil,
			assert: func(t *testing.T, ann map[string]string) {
				require.Nil(t, ann)
			},
		},
		{
			name:    "rule with no structured fields yields only the rule",
			reasons: []dispatch.Reason{{Rule: "would-regress", Message: "held"}},
			assert: func(t *testing.T, ann map[string]string) {
				require.Equal(t, map[string]string{
					annotationKeyDispatchRules: "would-regress",
				}, ann)
			},
		},
		{
			name: "distinct rules, blocked-by, and soonest until",
			reasons: []dispatch.Reason{
				{Rule: "freezes", Until: &until},
				{Rule: "yield-to-rollback", BlockedBy: "rb.01", Until: &earlier},
			},
			assert: func(t *testing.T, ann map[string]string) {
				require.Equal(t, "freezes,yield-to-rollback", ann[annotationKeyDispatchRules])
				require.Equal(t, "rb.01", ann[annotationKeyDispatchBlockedBy])
				// The soonest self-clear wins.
				require.Equal(t, "2026-07-15T18:00:00Z", ann[annotationKeyDispatchUntil])
			},
		},
		{
			name: "duplicate rule collapses in the rules list",
			reasons: []dispatch.Reason{
				{Rule: "yield-to-rollback", BlockedBy: "rb.01"},
				{Rule: "yield-to-rollback", BlockedBy: "rb.02"},
			},
			assert: func(t *testing.T, ann map[string]string) {
				require.Equal(t, "yield-to-rollback", ann[annotationKeyDispatchRules])
				require.Equal(t, "rb.01,rb.02", ann[annotationKeyDispatchBlockedBy])
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, dispatchEventAnnotations(tc.reasons))
		})
	}
}

func TestConditionReasonForHeld(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		rules []string
		want  string
	}{
		{name: "single mapped rule yields its token", rules: []string{"freezes"}, want: "Frozen"},
		{name: "single yield rule", rules: []string{"yield-to-rollback"}, want: "YieldToRollback"},
		{
			name:  "multiple rules fall back",
			rules: []string{"freezes", "would-regress"},
			want:  conditionReasonDispatchBlocked,
		},
		{
			name:  "unmapped custom rule falls back",
			rules: []string{"pci-change-ticket"},
			want:  conditionReasonDispatchBlocked,
		},
		{name: "no rules fall back", rules: nil, want: conditionReasonDispatchBlocked},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			set := map[string]struct{}{}
			for _, r := range tc.rules {
				set[r] = struct{}{}
			}
			require.Equal(t, tc.want, conditionReasonForHeld(set))
		})
	}
}
