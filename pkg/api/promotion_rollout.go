package api

import (
	"slices"
	"strings"

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

// PromotionRolloutOf returns the id of the PromotionRollout the Promotion was
// created by, as one part of that rollout's fan-out to Targets, or an empty
// string if the Promotion is not the child of one. The id is the UID the API
// serves the rollout with. The link is the kargo.akuity.io/promotion-rollout
// annotation: a PromotionRollout is not a Kubernetes resource and so cannot
// own the Promotion.
func PromotionRolloutOf(promo *kargoapi.Promotion) string {
	return promo.Annotations[kargoapi.AnnotationKeyPromotionRollout]
}

// PromotionRolloutTargetStatuses derives, from the Promotions of a Stage,
// what has become of each Target of the PromotionRollout with the given id
// (its UID): the newest
// Promotion to it and that Promotion's phase, one entry per Target in Target
// name order, together with a tally of those phases. Only Promotions that
// carry the rollout's id and name a Target count. A Target promoted to more than once
// within the rollout, for instance after a retry, reports its newest
// Promotion: Promotion names embed a ULID, so among one Target's children
// name order is creation order. The summary is nil when no Target has a
// Promotion yet.
func PromotionRolloutTargetStatuses(
	rolloutID string,
	promos []kargoapi.Promotion,
) ([]kargoapi.PromotionRolloutTargetStatus, *kargoapi.PromotionRolloutSummary) {
	newest := map[string]*kargoapi.Promotion{}
	for i := range promos {
		promo := &promos[i]
		if promo.Spec.Target == "" || PromotionRolloutOf(promo) != rolloutID {
			continue
		}
		if current, ok := newest[promo.Spec.Target]; !ok || promo.Name > current.Name {
			newest[promo.Spec.Target] = promo
		}
	}
	if len(newest) == 0 {
		return nil, nil
	}
	statuses := make([]kargoapi.PromotionRolloutTargetStatus, 0, len(newest))
	summary := &kargoapi.PromotionRolloutSummary{}
	for target, promo := range newest {
		statuses = append(statuses, kargoapi.PromotionRolloutTargetStatus{
			Name:      target,
			Promotion: promo.Name,
			Phase:     promo.Status.Phase,
		})
		switch promo.Status.Phase {
		case kargoapi.PromotionPhaseRunning:
			summary.Running++
		case kargoapi.PromotionPhaseSucceeded:
			summary.Succeeded++
		case kargoapi.PromotionPhaseFailed:
			summary.Failed++
		case kargoapi.PromotionPhaseErrored:
			summary.Errored++
		case kargoapi.PromotionPhaseAborted:
			summary.Aborted++
		default:
			// A Promotion the controller has not picked up yet has no phase.
			// It is Pending in every sense that matters.
			summary.Pending++
		}
	}
	slices.SortFunc(statuses, func(a, b kargoapi.PromotionRolloutTargetStatus) int {
		return strings.Compare(a.Name, b.Name)
	})
	return statuses, summary
}
