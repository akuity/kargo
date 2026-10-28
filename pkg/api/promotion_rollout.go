package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GeneratePromotionRolloutName generates a name for a PromotionRollout by
// combining the Stage name, a ULID, and a short hash of the Freight, in the
// same format used for Promotion names. Callers that need a PromotionRollout
// name should always use this function.
func GeneratePromotionRolloutName(stageName, freight string) string {
	return generatePromotionStyleName(stageName, freight)
}

// NewPromotionRollout builds a PromotionRollout that promotes the named
// Freight to the Targets the Stage governs, on behalf of the Stage.
//
// stageTargets should contain all of the Stage's Targets. As for a
// PromotionRequest, they are recorded in the rollout's spec and any
// percentage in the Stage's update strategy is resolved against how many
// there are. That list is never recomputed, so the rollout's threshold and
// terminal state are computed against a fixed set rather than a selector
// that could match differently later. A Target that appears after the
// rollout is created is picked up by a subsequent PromotionRollout.
//
// The result carries no owner references and no labels: a PromotionRollout
// is not a Kubernetes resource. Its Stage is recorded in its spec, and the
// store it is written to records its Project.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRollout(
	stage *kargoapi.Stage,
	freightName string,
	stageTargets []kargoapi.Target,
) (*kargoapi.PromotionRollout, error) {
	// Never nil: spec.targets is a required field, and an empty list records
	// that the Stage governed no Targets at this moment.
	specTargets := make([]kargoapi.PromotionRolloutTarget, len(stageTargets))
	for i, target := range stageTargets {
		specTargets[i] = kargoapi.PromotionRolloutTarget{Name: target.Name}
	}

	var updateStrategy kargoapi.PromotionRolloutUpdateStrategy
	if stage.Spec.Targets != nil {
		strategy, err := calculateUpdateStrategy(
			stage.Spec.Targets.UpdateStrategy,
			len(stageTargets),
		)
		if err != nil {
			return nil, err
		}
		updateStrategy = kargoapi.PromotionRolloutUpdateStrategy(strategy)
	}

	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      GeneratePromotionRolloutName(stage.Name, freightName),
		},
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:          stage.Name,
			Freight:        freightName,
			Targets:        specTargets,
			UpdateStrategy: updateStrategy,
		},
	}, nil
}
