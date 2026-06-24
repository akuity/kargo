package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSetAutoPromotionHoldAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{Name: "p"}}
	SetAutoPromotionHoldAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold])
}

func TestSetAutoPromotionReleaseAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{Name: "p"}}
	SetAutoPromotionReleaseAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionRelease])
}

func TestAutoPromotionHoldOriginFromPromotion(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}

	t.Run("no annotation", func(t *testing.T) {
		_, ok := AutoPromotionHoldOriginFromPromotion(&kargoapi.Promotion{})
		require.False(t, ok)
	})

	t.Run("malformed annotation", func(t *testing.T) {
		promo := &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{kargoapi.AnnotationKeyAutoPromotionHold: "not/a/valid/key"},
			},
		}
		_, ok := AutoPromotionHoldOriginFromPromotion(promo)
		require.False(t, ok)
	})

	t.Run("valid annotation", func(t *testing.T) {
		promo := &kargoapi.Promotion{}
		SetAutoPromotionHoldAnnotation(promo, origin)
		got, ok := AutoPromotionHoldOriginFromPromotion(promo)
		require.True(t, ok)
		require.Equal(t, origin, got)
	})
}

func TestAutoPromotionReleaseOriginFromPromotion(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}

	t.Run("no annotation", func(t *testing.T) {
		_, ok := AutoPromotionReleaseOriginFromPromotion(&kargoapi.Promotion{})
		require.False(t, ok)
	})

	t.Run("valid annotation", func(t *testing.T) {
		promo := &kargoapi.Promotion{}
		SetAutoPromotionReleaseAnnotation(promo, origin)
		got, ok := AutoPromotionReleaseOriginFromPromotion(promo)
		require.True(t, ok)
		require.Equal(t, origin, got)
	})
}
