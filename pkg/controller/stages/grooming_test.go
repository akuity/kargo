package stages

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/indexer"
)

func TestRegularStageReconciler_groomPromotions(t *testing.T) {
	t.Parallel()

	const (
		project    = "demo"
		stageName  = "prod"
		freightA   = "freight-a"
		freightB   = "freight-b"
		userActor  = "user:alice"
		rollbackOn = kargoapi.AnnotationValueTrue
	)

	// Freight objects the groomer resolves origins from. freightA/freightB sit
	// in distinct origins so the multi-origin isolation case has two buckets.
	freightObjs := []client.Object{
		&kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Name: freightA, Namespace: project},
			Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "nginx"},
		},
		&kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Name: freightB, Namespace: project},
			Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "redis"},
		},
	}

	// originOf maps a Freight name to its canonical origin key.
	originOf := map[string]string{
		freightA: "Warehouse/nginx",
		freightB: "Warehouse/redis",
	}

	// promo builds a Promotion. class is one of "auto"/"manual"/"manual-hold"/
	// "rollback". "manual-hold" is a manual-forward that deliberately promoted a
	// non-candidate Freight, marked with the auto-promotion hold-intent
	// annotation the webhook stamps at creation -- the only manual G1b displaces
	// autos for. Plain "manual" promoted the current candidate (resume intent).
	promo := func(name, freight, class string, phase kargoapi.PromotionPhase) *kargoapi.Promotion {
		p := &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   project,
				Annotations: map[string]string{},
			},
			Spec:   kargoapi.PromotionSpec{Stage: stageName, Freight: freight},
			Status: kargoapi.PromotionStatus{Phase: phase},
		}
		switch class {
		case "manual":
			p.Annotations[kargoapi.AnnotationKeyCreateActor] = userActor
		case "manual-hold":
			p.Annotations[kargoapi.AnnotationKeyCreateActor] = userActor
			p.Annotations[kargoapi.AnnotationKeyAutoPromotionHold] = originOf[freight]
		case "rollback":
			p.Annotations[kargoapi.AnnotationKeyRollback] = rollbackOn
			p.Annotations[kargoapi.AnnotationKeyCreateActor] = userActor
		case "auto":
			// No create-actor annotation -> auto-forward.
		}
		return p
	}

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: stageName, Namespace: project},
	}

	testCases := []struct {
		name   string
		promos []*kargoapi.Promotion
		// wantSupersededBy maps a Promotion name to the SupersededBy value it
		// should carry after grooming. Names absent from the map must NOT be
		// superseded.
		wantSupersededBy map[string]string
	}{
		{
			name: "G1 keeps the newest auto per origin",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-02", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-03", freightA, "auto", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{
				"auto-01": "auto-03",
				"auto-02": "auto-03",
			},
		},
		{
			name: "single auto is left alone",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{},
		},
		{
			name: "origins are groomed independently",
			promos: []*kargoapi.Promotion{
				promo("auto-a1", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-a2", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-b1", freightB, "auto", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{
				"auto-a1": "auto-a2",
			},
		},
		{
			name: "G1b: a pending non-candidate manual displaces every competing auto",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-02", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("man-09", freightA, "manual-hold", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{
				"auto-01": "man-09",
				"auto-02": "man-09",
			},
		},
		{
			name: "G1b: a running non-candidate manual displaces competing autos",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("man-09", freightA, "manual-hold", kargoapi.PromotionPhaseRunning),
			},
			wantSupersededBy: map[string]string{
				"auto-01": "man-09",
			},
		},
		{
			name: "a manual promoting the candidate does not displace autos (G1 only)",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhasePending),
				promo("auto-02", freightA, "auto", kargoapi.PromotionPhasePending),
				// No hold-intent: promoted the current candidate, so G1b must not
				// fire. G1 still coalesces the autos to the newest.
				promo("man-09", freightA, "manual", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{
				"auto-01": "auto-02",
			},
		},
		{
			name: "manual and rollback promotions are never superseded",
			promos: []*kargoapi.Promotion{
				promo("man-01", freightA, "manual", kargoapi.PromotionPhasePending),
				promo("man-02", freightA, "manual", kargoapi.PromotionPhasePending),
				promo("rb-01", freightA, "rollback", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{},
		},
		{
			name: "terminal autos are ignored",
			promos: []*kargoapi.Promotion{
				promo("auto-01", freightA, "auto", kargoapi.PromotionPhaseSucceeded),
				promo("auto-02", freightA, "auto", kargoapi.PromotionPhasePending),
			},
			wantSupersededBy: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			objects := append([]client.Object{}, freightObjs...)
			for _, p := range tc.promos {
				objects = append(objects, p)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
				Build()

			r := &RegularStageReconciler{client: c}

			_, err := r.groomPromotions(t.Context(), stage)
			require.NoError(t, err)

			for _, p := range tc.promos {
				got := &kargoapi.Promotion{}
				require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(p), got))
				req, ok := api.SupersedePromotionAnnotationValue(got.GetAnnotations())
				if want, expected := tc.wantSupersededBy[p.Name]; expected {
					require.True(t, ok, "Promotion %q should have been superseded", p.Name)
					require.Equal(t, want, req.SupersededBy)
				} else {
					require.False(t, ok, "Promotion %q should NOT have been superseded", p.Name)
				}
			}
		})
	}
}

func TestRegularStageReconciler_groomPromotions_skipsCurrentPromotion(t *testing.T) {
	t.Parallel()

	const project, stageName = "demo", "prod"

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: "freight-a", Namespace: project},
		Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "nginx"},
	}
	autoPromo := func(name string) *kargoapi.Promotion {
		return &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: project, Annotations: map[string]string{}},
			Spec:       kargoapi.PromotionSpec{Stage: stageName, Freight: "freight-a"},
			Status:     kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhasePending},
		}
	}
	// The gate selected the OLDER auto for dispatch; grooming must leave it be
	// and coalesce only among the rest.
	current, mid, newest := autoPromo("auto-01"), autoPromo("auto-02"), autoPromo("auto-03")

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(freight, current, mid, newest).
		WithIndex(&kargoapi.Promotion{}, indexer.PromotionsByStageField, indexer.PromotionsByStage).
		Build()

	r := &RegularStageReconciler{client: c}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: stageName, Namespace: project},
		Status: kargoapi.StageStatus{
			CurrentPromotion: &kargoapi.PromotionReference{Name: "auto-01"},
		},
	}

	_, err := r.groomPromotions(t.Context(), stage)
	require.NoError(t, err)

	assertSuperseded := func(name string, wantBy string) {
		got := &kargoapi.Promotion{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Namespace: project, Name: name}, got))
		req, ok := api.SupersedePromotionAnnotationValue(got.GetAnnotations())
		if wantBy == "" {
			require.False(t, ok, "%q should not be superseded", name)
			return
		}
		require.True(t, ok, "%q should be superseded", name)
		require.Equal(t, wantBy, req.SupersededBy)
	}
	assertSuperseded("auto-01", "")        // the dispatched candidate is untouched
	assertSuperseded("auto-02", "auto-03") // coalesced among the remainder
	assertSuperseded("auto-03", "")        // newest of the remainder survives
}

func TestRegularStageReconciler_groomPromotions_idempotent(t *testing.T) {
	t.Parallel()

	const project, stageName = "demo", "prod"

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: "freight-a", Namespace: project},
		Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "nginx"},
	}

	// An older auto that already carries the supersede intent (naming a prior
	// winner) must be left untouched, not re-stamped.
	preStamped := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auto-01",
			Namespace: project,
			Annotations: map[string]string{
				kargoapi.AnnotationKeySupersede: (&kargoapi.SupersedePromotionRequest{
					SupersededBy: "auto-02",
				}).String(),
			},
		},
		Spec:   kargoapi.PromotionSpec{Stage: stageName, Freight: "freight-a"},
		Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhasePending},
	}
	newest := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{Name: "auto-03", Namespace: project, Annotations: map[string]string{}},
		Spec:       kargoapi.PromotionSpec{Stage: stageName, Freight: "freight-a"},
		Status:     kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhasePending},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(freight, preStamped, newest).
		WithIndex(&kargoapi.Promotion{}, indexer.PromotionsByStageField, indexer.PromotionsByStage).
		Build()

	r := &RegularStageReconciler{client: c}
	stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Name: stageName, Namespace: project}}

	_, err := r.groomPromotions(t.Context(), stage)
	require.NoError(t, err)

	// The pre-stamped intent is preserved verbatim (not overwritten to auto-03).
	got := &kargoapi.Promotion{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(preStamped), got))
	req, ok := api.SupersedePromotionAnnotationValue(got.GetAnnotations())
	require.True(t, ok)
	require.Equal(t, "auto-02", req.SupersededBy)

	// The newest auto remains un-superseded.
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(newest), got))
	_, ok = api.SupersedePromotionAnnotationValue(got.GetAnnotations())
	require.False(t, ok)
}
