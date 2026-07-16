package targetpromotion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestBatches(t *testing.T) {
	promotions := []kargoapi.Promotion{
		newPromotion("", "", kargoapi.PromotionPhaseSucceeded, time.Unix(1, 0)),
		newPromotion("target", "", kargoapi.PromotionPhaseSucceeded, time.Unix(2, 0)),
		newPromotion("one", "first", kargoapi.PromotionPhaseSucceeded, time.Unix(3, 0)),
		newPromotion("two", "first", kargoapi.PromotionPhaseSucceeded, time.Unix(4, 0)),
		newPromotion("three", "second", kargoapi.PromotionPhaseSucceeded, time.Unix(5, 0)),
	}

	batches := Batches(promotions)
	require.Len(t, batches, 2)

	mostRecent := MostRecent(batches)
	require.NotNil(t, mostRecent)
	require.Equal(t, "freight", mostRecent.FreightName())
	require.Equal(t, StateSucceeded, mostRecent.State())
}

func TestBatchState(t *testing.T) {
	testCases := []struct {
		name       string
		promotions []kargoapi.Promotion
		expected   State
	}{
		{
			name: "active",
			promotions: []kargoapi.Promotion{
				newPromotion("one", "batch", kargoapi.PromotionPhaseRunning, time.Time{}),
			},
			expected: StateActive,
		},
		{
			name: "failed",
			promotions: []kargoapi.Promotion{
				newPromotion("one", "batch", kargoapi.PromotionPhaseSucceeded, time.Time{}),
				newPromotion("two", "batch", kargoapi.PromotionPhaseFailed, time.Time{}),
			},
			expected: StateFailed,
		},
		{
			name: "succeeded",
			promotions: []kargoapi.Promotion{
				newPromotion("one", "batch", kargoapi.PromotionPhaseSucceeded, time.Time{}),
				newPromotion("two", "batch", kargoapi.PromotionPhaseSucceeded, time.Time{}),
			},
			expected: StateSucceeded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			batch := MostRecent(Batches(testCase.promotions))
			require.NotNil(t, batch)
			require.Equal(t, testCase.expected, batch.State())
		})
	}
}

func TestBatchHasTarget(t *testing.T) {
	batch := MostRecent(Batches([]kargoapi.Promotion{
		newPromotion("one", "batch", kargoapi.PromotionPhaseSucceeded, time.Time{}),
		newPromotion("two", "batch", kargoapi.PromotionPhaseSucceeded, time.Time{}),
	}))
	require.NotNil(t, batch)

	require.True(t, batch.HasTarget("one"))
	require.False(t, batch.HasTarget("three"))
}

func newPromotion(
	target string,
	batch string,
	phase kargoapi.PromotionPhase,
	createdAt time.Time,
) kargoapi.Promotion {
	return kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(createdAt),
			Labels: map[string]string{
				kargoapi.LabelKeyPromotionBatch: batch,
			},
		},
		Spec: kargoapi.PromotionSpec{Freight: "freight", Target: target},
		Status: kargoapi.PromotionStatus{
			Phase: phase,
		},
	}
}
