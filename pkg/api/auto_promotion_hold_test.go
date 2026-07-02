package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSetAutoPromotionHoldAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyAutoPromotionResume: "Warehouse/old",
			},
		},
	}
	SetAutoPromotionHoldAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold])
	require.NotContains(t, promo.Annotations, kargoapi.AnnotationKeyAutoPromotionResume)
}

func TestSetAutoPromotionResumeAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/old",
			},
		},
	}
	SetAutoPromotionResumeAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume])
	require.NotContains(t, promo.Annotations, kargoapi.AnnotationKeyAutoPromotionHold)
}
