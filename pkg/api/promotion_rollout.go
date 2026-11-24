package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// GenerateRolloutChildPromotionName generates the name of the Promotion a
// PromotionRollout creates to promote its Freight to one of its Targets.
//
// The name has the format of:
//
//	<stage-name>.<target-name>.<rollout-id>
//
// Unlike every other generated Promotion name, it carries no ULID. A rollout
// creates at most one Promotion per Target, so the rollout's id and the
// Target's name identify the child completely, and creating it a second time
// -- from a stale view of the rollout's children, or from a second controller
// instance -- fails with AlreadyExists instead of running the same promotion
// twice. The short Freight hash is left out as well: the rollout fixes the
// Freight.
//
// Lex order of these names is therefore not creation order. Only
// ComparePromotionByPhaseAndCreationTime reads it that way, and the regular
// Stage reconciler that sorts with it never handles a target-aware Stage.
//
// A name too long for a Kubernetes resource keeps as much of
// <stage-name>.<target-name> as fits and ends it with a short hash of the
// whole of it:
//
//	<truncated-stage-and-target>-<hash>.<rollout-id>
//
// Truncating alone would cut the Target's name, which is the end of the
// prefix, and two Targets of one rollout could then be given the same name.
// The second create would fail with AlreadyExists, be taken for a child that
// already exists, and that Target would never be promoted to. The hash keeps
// the names of distinct Targets distinct. The rollout id is part of the name,
// so the hash only has to tell apart the Targets of one rollout.
//
// WARNING: Do not change how these names are generated. A rollout recognizes
// a child it has already created by this name. Under a different scheme, a
// child created by an earlier version would not match, and its Target could
// be promoted to twice.
func GenerateRolloutChildPromotionName(stageName, targetName, rolloutID string) string {
	if stageName == "" || targetName == "" || rolloutID == "" {
		return ""
	}
	prefix := stageName + promotionNameSeparator + targetName
	// A rollout id is a UUID, so the limit is always well above the length
	// of a short hash.
	limit := maxResourceNameLength - len(promotionNameSeparator) - len(rolloutID)
	if len(prefix) > limit {
		sum := sha256.Sum256([]byte(prefix))
		hash := hex.EncodeToString(sum[:])[:rolloutChildHashLength]
		// Truncation can land on a separator or hyphen. Either would put two
		// separators side by side, or start a part of the name with a hyphen,
		// and a Kubernetes resource name may do neither.
		kept := strings.TrimRight(
			prefix[:limit-len(rolloutChildHashSeparator)-len(hash)],
			promotionNameSeparator+rolloutChildHashSeparator,
		)
		prefix = kept + rolloutChildHashSeparator + hash
	}
	return strings.ToLower(prefix + promotionNameSeparator + rolloutID)
}

const (
	// maxResourceNameLength is the longest name a Kubernetes resource may
	// have.
	maxResourceNameLength = 253

	// rolloutChildHashSeparator separates the truncated part of an over-long
	// rollout child's name from the hash that keeps it unique.
	rolloutChildHashSeparator = "-"

	// rolloutChildHashLength is the length of that hash.
	//
	// WARNING: Do not change this. See GenerateRolloutChildPromotionName.
	rolloutChildHashLength = 8
)

// NewPromotionRolloutForAllTargets builds a PromotionRollout that promotes the named
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
func NewPromotionRolloutForAllTargets(
	stage *kargoapi.Stage,
	freightName string,
	stageTargets []kargoapi.Target,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRollout, error) {

	promotionUpdateStrategy, err := precheckPromotionRolloutFields(stage, stageTargets, updateStrategy)
	if err != nil {
		return nil, err
	}

	// Never nil: spec.targets is a required field, and an empty list records
	// that the Stage governed no Targets at this moment.
	rolloutTargets := make([]kargoapi.PromotionRolloutTarget, len(stageTargets))
	for i, target := range stageTargets {
		rolloutTargets[i] = kargoapi.PromotionRolloutTarget{Name: target.Name}
	}

	return newPromotionRollout(stage, freightName, rolloutTargets, *promotionUpdateStrategy), nil
}

// NewPromotionRolloutForSelectTargets builds a PromotionRollout that promotes the named
// Freight to the selected rolloutTargets Targets, on behalf of the Stage.
//
// stageTargets should contain all of the Stage's Targets. As for a
// PromotionRequest, they are recorded in the rollout's spec and any
// percentage in the Stage's update strategy is resolved against how many
// there are. That list is never recomputed, so the rollout's threshold and
// terminal state are computed against a fixed set rather than a selector
// that could match differently later. A Target that appears after the
// rollout is created is picked up by a subsequent PromotionRollout.
//
// rolloutTargets should contain names, which are a subset of stageTargets.
//
// The Stage MUST be target-aware, if it's not, the error is returned.
func NewPromotionRolloutForSelectTargets(
	stage *kargoapi.Stage,
	freightName string,
	stageTargets []kargoapi.Target,
	selectTargets []string,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRollout, error) {

	promotionUpdateStrategy, err := precheckPromotionRolloutFields(stage, stageTargets, updateStrategy)
	if err != nil {
		return nil, err
	}

	// Validate that all requested targets are selected by the stage
	stageTargetsNameSet := make(map[string]any, len(stageTargets))
	for _, stageTarget := range stageTargets {
		stageTargetsNameSet[stageTarget.Name] = struct{}{}
	}

	selectTargetsNameSet := make(map[string]any, len(selectTargets))

	rolloutTargets := []kargoapi.PromotionRolloutTarget{}
	for _, selectTarget := range selectTargets {
		if _, ok := stageTargetsNameSet[selectTarget]; !ok {
			return nil, fmt.Errorf(
				"%w: target %q is not selected by stage %q",
				ErrTargetNotSelected,
				selectTarget,
				stage.Name)
		}
		// Deduplicate configured targets
		if _, ok := selectTargetsNameSet[selectTarget]; !ok {
			selectTargetsNameSet[selectTarget] = struct{}{}
			rolloutTargets = append(rolloutTargets, kargoapi.PromotionRolloutTarget{Name: selectTarget})
		}
	}

	return newPromotionRollout(stage, freightName, rolloutTargets, *promotionUpdateStrategy), nil
}

func newPromotionRollout(
	stage *kargoapi.Stage,
	freightName string,
	rolloutTargets []kargoapi.PromotionRolloutTarget,
	updateStrategy kargoapi.PromotionRolloutUpdateStrategy,
) *kargoapi.PromotionRollout {
	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      GeneratePromotionRolloutName(stage.Name, freightName),
		},
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:          stage.Name,
			Freight:        freightName,
			Targets:        rolloutTargets,
			UpdateStrategy: updateStrategy,
		},
	}
}

// precheckPromotionRolloutFields initializes shared fields and runs shared checks
// for different NewPromotionRollout functions
func precheckPromotionRolloutFields(
	stage *kargoapi.Stage,
	stageTargets []kargoapi.Target,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRolloutUpdateStrategy, error) {
	if stage == nil {
		return nil, fmt.Errorf("cannot create promotion request without a stage")
	}
	if stage.Spec.Targets == nil {
		return nil, fmt.Errorf("cannot create promotion request for a stage without targets")
	}

	promotionUpdateStrategy, err := ensureRolloutUpdateStrategy(updateStrategy, stage, stageTargets)
	if err != nil {
		return nil, err
	}
	return promotionUpdateStrategy, nil
}

func ensureRolloutUpdateStrategy(
	updateStrategy *kargoapi.TargetUpdateStrategy,
	stage *kargoapi.Stage,
	stageTargets []kargoapi.Target,
) (*kargoapi.PromotionRolloutUpdateStrategy, error) {
	if updateStrategy == nil {
		return calculateRolloutUpdateStrategy(stage.Spec.Targets.UpdateStrategy, len(stageTargets))
	}

	return calculateRolloutUpdateStrategy(*updateStrategy, len(stageTargets))
}

func calculateRolloutUpdateStrategy(
	targetUpdateStrategy kargoapi.TargetUpdateStrategy,
	numTargets int,
) (*kargoapi.PromotionRolloutUpdateStrategy, error) {
	strategy, err := CalculateUpdateStrategy(targetUpdateStrategy, numTargets)
	if err != nil {
		return nil, err
	}
	return new(kargoapi.PromotionRolloutUpdateStrategy(*strategy)), nil
}
