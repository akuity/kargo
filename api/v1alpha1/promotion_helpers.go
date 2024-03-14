package v1alpha1

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPromotion returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Promotion, error) {
	promo := Promotion{}
	if err := c.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Promotion %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &promo, nil
}

// RefreshPromotion forces reconciliation of a Promotion by setting an annotation
// on the Promotion, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Promotion, error) {
	promo := &Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(ctx, c, promo, AnnotationKeyRefresh, time.Now().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return promo, nil
}

// ClearPromotionRefresh is called by the Promotion controller to clear the refresh
// annotation on the Promotion (if present).
func ClearPromotionRefresh(
	ctx context.Context,
	c client.Client,
	promo *Promotion,
) error {
	if promo.Annotations == nil {
		return nil
	}
	if _, ok := promo.Annotations[AnnotationKeyRefresh]; !ok {
		return nil
	}
	newPromo := Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promo.Name,
			Namespace: promo.Namespace,
		},
	}
	return clearObjectAnnotation(ctx, c, &newPromo, AnnotationKeyRefresh)
}
