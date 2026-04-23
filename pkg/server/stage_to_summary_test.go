package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestStageToSummary(t *testing.T) {
	t.Run("nil stage returns nil", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, stageToSummary(nil))
	})

	t.Run("minimal stage preserves identity and zero-valued fields", func(t *testing.T) {
		t.Parallel()
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "dev", Namespace: "proj"},
		}
		got := stageToSummary(s)
		require.NotNil(t, got)
		require.NotNil(t, got.Metadata)
		require.Equal(t, "dev", got.Metadata.Name)
		require.Equal(t, "proj", got.Metadata.Namespace)
		require.NotNil(t, got.Spec)
		require.Empty(t, got.Spec.Shard)
		require.Empty(t, got.Spec.RequestedFreight)
		require.Zero(t, got.Spec.PromotionStepCount)
		require.False(t, got.Spec.HasVerification)
		require.NotNil(t, got.Status)
		require.Empty(t, got.Status.Conditions)
		require.Nil(t, got.Status.CurrentFreight)
		require.Nil(t, got.Status.CurrentPromotion)
		require.Nil(t, got.Status.LastPromotion)
		require.Nil(t, got.Status.Health)
		require.Zero(t, got.Status.ObservedGeneration)
		require.False(t, got.Status.AutoPromotionEnabled)
	})

	t.Run("fully populated stage projects every summary field", func(t *testing.T) {
		t.Parallel()
		health := &kargoapi.Health{Status: kargoapi.HealthStateHealthy}
		currentPromo := &kargoapi.PromotionReference{Name: "promo-current"}
		lastPromo := &kargoapi.PromotionReference{Name: "promo-last"}

		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "qa",
				Namespace:   "proj",
				UID:         "uid-1",
				Labels:      map[string]string{"kargo.akuity.io/shard": "east"},
				Annotations: map[string]string{"kargo.akuity.io/description": "QA"},
			},
			Spec: kargoapi.StageSpec{
				Shard: "east",
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh-1"},
					Sources: kargoapi.FreightSources{Direct: true},
				}},
				PromotionTemplate: &kargoapi.PromotionTemplate{
					Spec: kargoapi.PromotionTemplateSpec{
						Steps: []kargoapi.PromotionStep{{}, {}, {}},
					},
				},
				Verification: &kargoapi.Verification{},
			},
			Status: kargoapi.StageStatus{
				Conditions: []metav1.Condition{{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Healthy",
					Message: "all good",
				}},
				LastHandledRefresh: "refresh-token-1",
				FreightHistory: kargoapi.FreightHistory{
					{ID: "fc-current"},
					{ID: "fc-older"},
				},
				CurrentPromotion:     currentPromo,
				LastPromotion:        lastPromo,
				Health:               health,
				ObservedGeneration:   7,
				AutoPromotionEnabled: true,
				// Fields that must NOT appear in the summary:
				FreightSummary: "human-readable summary",
			},
		}

		got := stageToSummary(s)
		require.NotNil(t, got)

		// Metadata
		require.Equal(t, "qa", got.Metadata.Name)
		require.Equal(t, "proj", got.Metadata.Namespace)
		require.Equal(t, "east", got.Metadata.Labels["kargo.akuity.io/shard"])
		require.Equal(t, "QA", got.Metadata.Annotations["kargo.akuity.io/description"])

		// Spec summary
		require.Equal(t, "east", got.Spec.Shard)
		require.Len(t, got.Spec.RequestedFreight, 1)
		require.Equal(t, "wh-1", got.Spec.RequestedFreight[0].Origin.Name)
		require.Equal(t, int32(3), got.Spec.PromotionStepCount)
		require.True(t, got.Spec.HasVerification)

		// Status summary
		require.Len(t, got.Status.Conditions, 1)
		require.Equal(t, "Ready", got.Status.Conditions[0].Type)
		require.Equal(t, "refresh-token-1", got.Status.LastHandledRefresh)
		require.NotNil(t, got.Status.CurrentFreight)
		require.Equal(t, "fc-current", got.Status.CurrentFreight.ID)
		require.Equal(t, "promo-current", got.Status.CurrentPromotion.Name)
		require.Equal(t, "promo-last", got.Status.LastPromotion.Name)
		require.Equal(t, kargoapi.HealthStateHealthy, got.Status.Health.Status)
		require.Equal(t, int64(7), got.Status.ObservedGeneration)
		require.True(t, got.Status.AutoPromotionEnabled)
	})

	t.Run("empty freight history omits current freight", func(t *testing.T) {
		t.Parallel()
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "s"},
			Status:     kargoapi.StageStatus{FreightHistory: nil},
		}
		got := stageToSummary(s)
		require.NotNil(t, got)
		require.Nil(t, got.Status.CurrentFreight)
	})

	t.Run("nil promotion template yields zero step count", func(t *testing.T) {
		t.Parallel()
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "s"},
			Spec:       kargoapi.StageSpec{PromotionTemplate: nil},
		}
		got := stageToSummary(s)
		require.NotNil(t, got)
		require.Zero(t, got.Spec.PromotionStepCount)
	})

	t.Run("nil verification yields hasVerification false", func(t *testing.T) {
		t.Parallel()
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "s"},
			Spec:       kargoapi.StageSpec{Verification: nil},
		}
		got := stageToSummary(s)
		require.NotNil(t, got)
		require.False(t, got.Spec.HasVerification)
	})

	t.Run("mutating the summary does not affect the source Stage", func(t *testing.T) {
		t.Parallel()
		s := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "s", Labels: map[string]string{"k": "v"}},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{Name: "wh"},
				}},
			},
			Status: kargoapi.StageStatus{
				Conditions: []metav1.Condition{{Type: "Ready"}},
			},
		}
		got := stageToSummary(s)

		// Mutate the summary.
		got.Metadata.Labels["k"] = "mutated"
		got.Spec.RequestedFreight[0].Origin.Name = "mutated"
		got.Status.Conditions[0].Type = "Mutated"

		// Source Stage should be unchanged.
		require.Equal(t, "v", s.Labels["k"])
		require.Equal(t, "wh", s.Spec.RequestedFreight[0].Origin.Name)
		require.Equal(t, "Ready", s.Status.Conditions[0].Type)
	})
}
