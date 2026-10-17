package stages

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

func TestRemainingSoakForFreight(t *testing.T) {
	const required = time.Hour

	// recentlyEntered returns a Freight that has been in `stages` since `since`
	// without any prior completed soak.
	recentlyEntered := func(since time.Time, stages ...string) *kargoapi.Freight {
		f := &kargoapi.Freight{
			Status: kargoapi.FreightStatus{
				VerifiedIn:  map[string]kargoapi.VerifiedStage{},
				CurrentlyIn: map[string]kargoapi.CurrentStage{},
			},
		}
		for _, s := range stages {
			f.Status.VerifiedIn[s] = kargoapi.VerifiedStage{}
			f.Status.CurrentlyIn[s] = kargoapi.CurrentStage{
				Since: &metav1.Time{Time: since},
			}
		}
		return f
	}

	now := time.Now()

	testCases := []struct {
		name     string
		freight  *kargoapi.Freight
		stages   []string
		required time.Duration
		strategy kargoapi.FreightAvailabilityStrategy
		expect   func(t *testing.T, remaining time.Duration)
	}{
		{
			name:     "OneOf with single stage still soaking",
			freight:  recentlyEntered(now.Add(-10*time.Minute), "a"),
			stages:   []string{"a"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyOneOf,
			expect: func(t *testing.T, d time.Duration) {
				// Should be roughly 50 minutes remaining.
				assert.InDelta(t, 50*time.Minute, d, float64(2*time.Second))
			},
		},
		{
			name:     "OneOf returns minimum across stages",
			freight:  recentlyEntered(now.Add(-50*time.Minute), "a", "b"),
			stages:   []string{"a", "b"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyOneOf,
			expect: func(t *testing.T, d time.Duration) {
				// Both stages have been soaking the same amount; ~10m remaining.
				assert.InDelta(t, 10*time.Minute, d, float64(2*time.Second))
			},
		},
		{
			name: "OneOf returns 0 when any stage already satisfies soak",
			freight: func() *kargoapi.Freight {
				// One stage has a completed soak that exceeds required; another
				// is still soaking.
				f := recentlyEntered(now.Add(-1*time.Minute), "b")
				f.Status.VerifiedIn["a"] = kargoapi.VerifiedStage{
					LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
				}
				return f
			}(),
			stages:   []string{"a", "b"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyOneOf,
			expect: func(t *testing.T, d time.Duration) {
				assert.Equal(t, time.Duration(0), d)
			},
		},
		{
			name:     "OneOf skips Stages where Freight is not verified",
			freight:  recentlyEntered(now.Add(-30*time.Minute), "a"),
			stages:   []string{"a", "unverified"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyOneOf,
			expect: func(t *testing.T, d time.Duration) {
				assert.InDelta(t, 30*time.Minute, d, float64(2*time.Second))
			},
		},
		{
			name:     "All returns max across stages",
			freight:  recentlyEntered(now.Add(-30*time.Minute), "a", "b"),
			stages:   []string{"a", "b"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyAll,
			expect: func(t *testing.T, d time.Duration) {
				// Both at ~30m elapsed; max remaining is ~30m.
				assert.InDelta(t, 30*time.Minute, d, float64(2*time.Second))
			},
		},
		{
			name: "All ignores already-soaked stages and reports remaining for others",
			freight: func() *kargoapi.Freight {
				f := recentlyEntered(now.Add(-20*time.Minute), "b")
				// Stage "a" has already completed its soak.
				f.Status.VerifiedIn["a"] = kargoapi.VerifiedStage{
					LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
				}
				return f
			}(),
			stages:   []string{"a", "b"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyAll,
			expect: func(t *testing.T, d time.Duration) {
				// "a" done; "b" needs ~40m more.
				assert.InDelta(t, 40*time.Minute, d, float64(2*time.Second))
			},
		},
		{
			name:     "All returns 0 when Freight is not verified in every Stage",
			freight:  recentlyEntered(now.Add(-30*time.Minute), "a"),
			stages:   []string{"a", "b"},
			required: required,
			strategy: kargoapi.FreightAvailabilityStrategyAll,
			expect: func(t *testing.T, d time.Duration) {
				assert.Equal(t, time.Duration(0), d)
			},
		},
		{
			name:     "empty strategy is treated as OneOf",
			freight:  recentlyEntered(now.Add(-45*time.Minute), "a"),
			stages:   []string{"a"},
			required: required,
			strategy: "",
			expect: func(t *testing.T, d time.Duration) {
				assert.InDelta(t, 15*time.Minute, d, float64(2*time.Second))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			remaining := remainingSoakForFreight(
				tc.freight, tc.stages, tc.required, tc.strategy,
			)
			tc.expect(t, remaining)
		})
	}
}

func TestCalculateNextSoakCheck(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	const ns = "fake-project"
	const wh = "fake-warehouse"

	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: wh},
	}

	buildClient := func(objs ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithIndex(
				&kargoapi.Freight{},
				indexer.FreightByWarehouseField,
				indexer.FreightByWarehouse,
			).
			WithIndex(
				&kargoapi.Freight{},
				indexer.FreightByCurrentStagesField,
				indexer.FreightByCurrentStages,
			).
			WithIndex(
				&kargoapi.Freight{},
				indexer.FreightByVerifiedStagesField,
				indexer.FreightByVerifiedStages,
			).
			WithIndex(
				&kargoapi.Freight{},
				indexer.FreightApprovedForStagesField,
				indexer.FreightApprovedForStages,
			).
			Build()
	}

	stage := func(strategy kargoapi.FreightAvailabilityStrategy, soak time.Duration, sources ...string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "downstream"},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: wh,
						},
						Sources: kargoapi.FreightSources{
							Stages:               sources,
							RequiredSoakTime:     &metav1.Duration{Duration: soak},
							AvailabilityStrategy: strategy,
						},
					},
				},
			},
		}
	}

	freight := func(name string, since time.Time, verifiedIn ...string) *kargoapi.Freight {
		f := &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: wh,
			},
			Status: kargoapi.FreightStatus{
				VerifiedIn:  map[string]kargoapi.VerifiedStage{},
				CurrentlyIn: map[string]kargoapi.CurrentStage{},
			},
		}
		for _, s := range verifiedIn {
			f.Status.VerifiedIn[s] = kargoapi.VerifiedStage{}
			f.Status.CurrentlyIn[s] = kargoapi.CurrentStage{
				Since: &metav1.Time{Time: since},
			}
		}
		return f
	}

	now := time.Now()

	t.Run("no soak time configured returns 0", func(t *testing.T) {
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "downstream"},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: wh,
					},
					Sources: kargoapi.FreightSources{Stages: []string{"upstream"}},
				}},
			},
		}
		c := buildClient(warehouse, freight("f", now.Add(-10*time.Minute), "upstream"))
		d, err := calculateNextSoakCheck(t.Context(), c, s)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
	})

	t.Run("missing warehouse is skipped", func(t *testing.T) {
		s := stage(kargoapi.FreightAvailabilityStrategyOneOf, time.Hour, "upstream")
		c := buildClient() // no warehouse, no freight
		d, err := calculateNextSoakCheck(t.Context(), c, s)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
	})

	t.Run("returns soonest deadline across candidates with buffer", func(t *testing.T) {
		s := stage(kargoapi.FreightAvailabilityStrategyOneOf, time.Hour, "upstream")
		c := buildClient(
			warehouse,
			// 45 minutes elapsed -> ~15 minutes remaining (this should win)
			freight("f1", now.Add(-45*time.Minute), "upstream"),
			// 10 minutes elapsed -> ~50 minutes remaining
			freight("f2", now.Add(-10*time.Minute), "upstream"),
		)
		d, err := calculateNextSoakCheck(t.Context(), c, s)
		require.NoError(t, err)
		assert.InDelta(t, 15*time.Minute+soakRequeueBuffer, d, float64(2*time.Second))
	})

	t.Run("returns 0 when every candidate already past soak", func(t *testing.T) {
		s := stage(kargoapi.FreightAvailabilityStrategyOneOf, time.Hour, "upstream")
		c := buildClient(
			warehouse,
			freight("done", now.Add(-2*time.Hour), "upstream"),
		)
		d, err := calculateNextSoakCheck(t.Context(), c, s)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
	})

	t.Run("All strategy returns max across upstream stages", func(t *testing.T) {
		s := stage(kargoapi.FreightAvailabilityStrategyAll, time.Hour, "a", "b")
		// Verified in both stages but with different "Since" times.
		f := &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "f"},
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: wh,
			},
			Status: kargoapi.FreightStatus{
				VerifiedIn: map[string]kargoapi.VerifiedStage{
					"a": {},
					"b": {},
				},
				CurrentlyIn: map[string]kargoapi.CurrentStage{
					"a": {Since: &metav1.Time{Time: now.Add(-45 * time.Minute)}},
					"b": {Since: &metav1.Time{Time: now.Add(-20 * time.Minute)}},
				},
			},
		}
		c := buildClient(warehouse, f)
		d, err := calculateNextSoakCheck(t.Context(), c, s)
		require.NoError(t, err)
		// "b" is the laggard with ~40m remaining.
		assert.InDelta(t, 40*time.Minute+soakRequeueBuffer, d, float64(2*time.Second))
	})
}
