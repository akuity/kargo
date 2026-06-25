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
