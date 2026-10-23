package api

import (
	"cmp"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GeneratePromotionRolloutName generates a name for a PromotionRollout by
// combining the Stage name, a ULID, and a short hash of the Freight, in the
// same format used for Promotion names. Callers that need a PromotionRollout
// name should always use this function.
func GeneratePromotionRolloutName(stageName, freight string) string {
	return generatePromotionStyleName(stageName, freight)
}

// ComparePromotionRolloutByPhaseAndNumber compares two PromotionRollouts by
// their phase and their number within their Stage. It returns a negative
// value if PromotionRollout `a` should come before PromotionRollout `b`, a
// positive value if `a` should come after `b`, or zero if they are considered
// equal for sorting purposes. It can be used in conjunction with
// slices.SortFunc to sort a list of PromotionRollouts.
//
// The order is the one ComparePromotionByPhaseAndCreationTime imposes on
// Promotions, so that a Stage chooses the PromotionRollout it is promoting
// through exactly as it chooses its current Promotion:
//
//  1. Running PromotionRollouts
//  2. Non-terminal PromotionRollouts (ordered by number in ascending order)
//  3. Terminal PromotionRollouts (ordered by number in descending order)
//
// A rollout's number is its position among its Stage's rollouts, so among the
// rollouts of one Stage number order is creation order. A rollout without a
// number (one built in memory rather than read from the API) is ordered by
// its name instead, which embeds a ULID and orders the same way.
func ComparePromotionRolloutByPhaseAndNumber(a, b kargoapi.PromotionRollout) int {
	// Compare the phases of the PromotionRollouts first.
	if phaseCompare := ComparePromotionRolloutPhase(
		a.Status.Phase,
		b.Status.Phase,
	); phaseCompare != 0 {
		return phaseCompare
	}

	creationOrder := cmp.Compare(a.Number, b.Number)
	if a.Number == 0 || b.Number == 0 {
		creationOrder = strings.Compare(a.Name, b.Name)
	}
	switch {
	case !a.Status.Phase.IsTerminal():
		// Non-terminal PromotionRollouts are ordered in ascending order, so that
		// the rollout which was (or will be) worked first is at the top.
		return creationOrder
	default:
		// Terminal PromotionRollouts are ordered in descending order, so that
		// the most recent rollout is at the top, limiting the number of rollouts
		// which have to be further inspected.
		return -creationOrder
	}
}

// ComparePromotionRolloutPhase compares two PromotionRollout phases. It
// returns a negative value if phase `a` should come before phase `b`, a
// positive value if phase `a` should come after phase `b`, or zero if they
// are considered equal for sorting purposes. It can be used in combination
// with slices.SortFunc to sort a list of PromotionRollout phases.
//
// The order of PromotionRollout phases matches the one ComparePromotionPhase
// imposes on Promotion phases:
//
//  1. Running
//  2. Non-terminal phases
//  3. Terminal phases
func ComparePromotionRolloutPhase(a, b kargoapi.PromotionRolloutPhase) int {
	aRunning := a == kargoapi.PromotionRolloutPhaseRunning
	bRunning := b == kargoapi.PromotionRolloutPhaseRunning
	aTerminal, bTerminal := a.IsTerminal(), b.IsTerminal()

	// NB: As in ComparePromotionPhase, the order of the cases here is important:
	// "Running" is a special case that should always come before any other phase.
	switch {
	case aRunning && !bRunning:
		return -1
	case !aRunning && bRunning:
		return 1
	case !aTerminal && bTerminal:
		return -1
	case aTerminal && !bTerminal:
		return 1
	default:
		return 0
	}
}

// NewPromotionRollout builds a PromotionRollout that promotes the named
// Freight to the Targets the Stage governs, on behalf of the Stage.
//
// Which Targets those are is not recorded here. The rollout reconciler
// resolves the Stage's selectors once, when it first sees the rollout, and
// records the result; that list is never recomputed, so the rollout's
// threshold and terminal state are computed against a fixed set rather than
// a selector that could match differently later. A Target that appears
// after that point is picked up by a subsequent PromotionRollout.
//
// The result carries no owner references and no labels: a PromotionRollout
// is not a Kubernetes resource. Its Stage is recorded in its spec, and the
// store it is written to records its Project.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRollout(
	stage *kargoapi.Stage,
	freightName string,
) *kargoapi.PromotionRollout {
	// The strategy is carried as the Stage gives it. A percentage counts the
	// rollout's Targets, which are resolved only later, so it is resolved
	// then too.
	updateStrategy := kargoapi.PromotionRolloutUpdateStrategy{}
	if stage.Spec.Targets != nil {
		updateStrategy.MaxConcurrent = copyIntOrString(stage.Spec.Targets.UpdateStrategy.MaxConcurrent)
		updateStrategy.MaxFailures = copyIntOrString(stage.Spec.Targets.UpdateStrategy.MaxFailures)
	}

	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      GeneratePromotionRolloutName(stage.Name, freightName),
		},
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:          stage.Name,
			Freight:        freightName,
			UpdateStrategy: updateStrategy,
		},
	}
}

// copyIntOrString returns a copy of the value, or nil for nil.
func copyIntOrString(value *intstr.IntOrString) *intstr.IntOrString {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
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
