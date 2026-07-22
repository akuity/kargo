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
