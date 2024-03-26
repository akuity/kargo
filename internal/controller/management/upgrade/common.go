package upgrade

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const v050CompatibilityLabelKey = "kargo.akuity.com/v0.5.0-compatible"

func ignoreDeletesPredicate() predicate.Funcs {
	return predicate.Funcs{
		DeleteFunc: func(event.DeleteEvent) bool {
			return false
		},
	}
}

func getNotV050CompatiblePredicate() (predicate.Predicate, error) {
	return predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      v050CompatibilityLabelKey,
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"true"},
				},
			},
		},
	)
}

func patchLabel(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	key string,
	value string,
) error {
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{"%s":"%s"}}}`,
			key,
			value,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	return c.Patch(ctx, obj, patch)
}
