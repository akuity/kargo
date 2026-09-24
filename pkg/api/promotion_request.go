package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// ComparePromotionRequestPhase compares two PromotionRequest phases. It returns
// a negative value if phase `a` should come before phase `b`, a positive value
// if phase `a` should come after phase `b`, or zero if they are considered
// equal for sorting purposes. It can be used in combination with
// slices.SortFunc to sort a list of PromotionRequest phases.
//
// The order of PromotionRequest phases matches the one ComparePromotionPhase
// imposes on Promotion phases:
//
//  1. Running
//  2. Non-terminal phases
//  3. Terminal phases
func ComparePromotionRequestPhase(a, b kargoapi.PromotionRequestPhase) int {
	aRunning := a == kargoapi.PromotionRequestPhaseRunning
	bRunning := b == kargoapi.PromotionRequestPhaseRunning
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

// NewPromotionRequest constructs a PromotionRequest expressing the intent to
// promote the given Freight to the given Targets: the ones the Stage governs
// at this moment, as resolved by FilterTargetsForStage.
//
// The list is recorded in spec.targets and never recomputed: a
// PromotionRequest is a snapshot of what the Stage governed when it was made,
// so its threshold and terminal state are computed against a fixed set rather
// than a selector that could match differently later. A Target that appears
// after this point is picked up by a subsequent PromotionRequest.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRequest(
	stage *kargoapi.Stage,
	freightName string,
	targets []kargoapi.Target,
) *kargoapi.PromotionRequest {
	// Never nil: spec.targets is a required field, so a nil slice would
	// serialize as null. An empty list is meaningful -- it records that the
	// Stage governed no Targets at this moment.
	specTargets := make([]kargoapi.PromotionRequestTarget, len(targets))
	for i, target := range targets {
		specTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
	}
	return &kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      GeneratePromotionRequestName(stage.Name, freightName),
			Labels:    map[string]string{kargoapi.LabelKeyStage: stage.Name},
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   stage.Name,
			Freight: freightName,
			Targets: specTargets,
		},
	}
}
