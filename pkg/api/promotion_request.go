package api

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GeneratePromotionRequestName generates a name for a PromotionRequest by
// combining the Stage name, a ULID, and a short hash of the Freight, in the
// same format used for Promotion names. Callers that need a PromotionRequest
// name should always use this function.
func GeneratePromotionRequestName(stageName, freight string) string {
	return generatePromotionStyleName(stageName, freight)
}

// GenerateChildPromotionName generates a name for a Promotion created by a
// PromotionRequest to promote Freight to one of its Targets.
//
// The name has the format of:
//
//	<stage-name>.<target-name>.<ulid>.<short-hash>
//
// Naming a child after its Target makes the fan-out legible at a glance and
// greppable by Target. The ULID and Freight hash are retained so that repeated
// promotions to the same Target do not collide, and so that lex order continues
// to match creation order as it does for every other Promotion.
func GenerateChildPromotionName(stageName, targetName, freight string) string {
	if stageName == "" || targetName == "" {
		return ""
	}
	return generatePromotionStyleName(
		stageName+promotionNameSeparator+targetName,
		freight,
	)
}

// NewPromotionRequest constructs a PromotionRequest expressing the intent to
// promote the given Freight to the Targets the Stage governs.
//
// The Stage's target selectors are resolved to concrete Targets here, once, and
// the result recorded in spec.targets. That list is never recomputed: a
// PromotionRequest is a snapshot of what the Stage governed at this moment, so
// its threshold and terminal state are computed against a fixed set rather than
// a selector that could match differently later. A Target that appears after
// this point is picked up by a subsequent PromotionRequest, not by re-resolving
// this one.
//
// The Stage is set as the controlling owner, so its PromotionRequests are
// garbage-collected with it.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRequest(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
	freightName string,
) (*kargoapi.PromotionRequest, error) {
	targets, err := ListTargetsForStage(ctx, c, stage)
	if err != nil {
		return nil, fmt.Errorf(
			"error resolving Targets governed by Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}

	// Never nil: spec.targets is a required field, so a nil slice would
	// serialize as null and be rejected. An empty list is meaningful -- it
	// records that the Stage governed no Targets at this moment.
	specTargets := make([]kargoapi.PromotionRequestTarget, len(targets))
	for i, target := range targets {
		specTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
	}

	labels := map[string]string{kargoapi.LabelKeyStage: stage.Name}
	// The PromotionRequest reconciler filters by shard, so a PromotionRequest
	// must carry the shard of the Stage that it promotes on behalf of. Without
	// this, only the default controller would ever reconcile it.
	if stage.Spec.Shard != "" {
		labels[kargoapi.LabelKeyShard] = stage.Spec.Shard
	}

	return &kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      GeneratePromotionRequestName(stage.Name, freightName),
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					stage,
					kargoapi.GroupVersion.WithKind("Stage"),
				),
			},
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   stage.Name,
			Freight: freightName,
			Targets: specTargets,
		},
	}, nil
}
