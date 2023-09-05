package kargo

import (
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

const (
	// maximum length of the stage name used in the promotion name prefix before it exceeds
	// kubernetes resource name limit of 253
	// 253 - 1 (.) - 26 (ulid) - 1 (.) - 7 (sha) = 218
	maxStageNamePrefixLength = 218
)

// NewPromotion returns a new Promotion from a given stage and freight with our naming convention.
// Ensures the owner reference is set to be the stage, and carries over any shard labels
func NewPromotion(stage kubev1alpha1.Stage, freight string) kubev1alpha1.Promotion {
	shortHash := freight
	if len(shortHash) > 7 {
		shortHash = freight[0:7]
	}
	shortStageName := stage.Name
	if len(stage.Name) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}

	// ulid.Make() is pseudo-random, not crypto-random, but we don't care.
	// We just want a unique ID that can be sorted lexicographically
	promoName := strings.ToLower(fmt.Sprintf("%s.%s.%s", shortStageName, ulid.Make(), shortHash))

	ownerRef := metav1.NewControllerRef(&stage, kubev1alpha1.GroupVersion.WithKind("Stage"))

	promotion := kubev1alpha1.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:            promoName,
			Namespace:       stage.Namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: &kubev1alpha1.PromotionSpec{
			Stage:   stage.Name,
			Freight: freight,
		},
	}
	if stage.Labels != nil && stage.Labels[controller.ShardLabelKey] != "" {
		promotion.ObjectMeta.Labels = map[string]string{
			controller.ShardLabelKey: stage.Labels[controller.ShardLabelKey],
		}
	}
	return promotion
}
