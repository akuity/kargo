package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var ErrTargetNotSelected = errors.New("target is not selected")

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

// ComparePromotionRequestByPhaseAndCreationTime compares two PromotionRequests
// by their phase and creation time. It returns a negative value if
// PromotionRequest `a` should come before PromotionRequest `b`, a positive
// value if `a` should come after `b`, or zero if they are considered equal for
// sorting purposes. It can be used in conjunction with slices.SortFunc to sort
// a list of PromotionRequests.
//
// The order is the one ComparePromotionByPhaseAndCreationTime imposes on
// Promotions, so that a Stage chooses the PromotionRequest it is promoting
// through exactly as it chooses its current Promotion:
//
//  1. Running PromotionRequests
//  2. Non-terminal PromotionRequests (ordered by ULID in ascending order)
//  3. Terminal PromotionRequests (ordered by ULID in descending order)
//
// As there, name order stands in for creation order: a generated
// PromotionRequest name is <stage>.<ulid>.<short-hash>, so among the requests
// of a single Stage everything left of the ULID is identical and comparing
// names whole is comparing the ULIDs.
func ComparePromotionRequestByPhaseAndCreationTime(a, b kargoapi.PromotionRequest) int {
	// Compare the phases of the PromotionRequests first.
	if phaseCompare := ComparePromotionRequestPhase(
		a.Status.Phase,
		b.Status.Phase,
	); phaseCompare != 0 {
		return phaseCompare
	}

	switch {
	case !a.Status.Phase.IsTerminal():
		// Non-terminal PromotionRequests are ordered in ascending order, so that
		// the request which was (or will be) worked first is at the top.
		return strings.Compare(a.Name, b.Name)
	default:
		// Terminal PromotionRequests are ordered in descending order, so that the
		// most recent request is at the top, limiting the number of requests which
		// have to be further inspected.
		return strings.Compare(b.Name, a.Name)
	}
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

// NewPromotionRequestForAllTargets creates a promotion request for all stage targets.
func NewPromotionRequestForAllTargets(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
	projectTargetsFn func(context.Context, string) ([]kargoapi.Target, error),
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRequest, error) {
	stageTargets, requestUpdateStrategy, err := precheckPromotionRequestFields(
		ctx, stage, projectTargetsFn, updateStrategy)
	if err != nil {
		return nil, err
	}

	// Never nil: spec.targets is a required field, so a nil slice would
	// serialize as null and be rejected. An empty list is meaningful -- it
	// records that the Stage governed no Targets at this moment.
	promotionRequestTargets := make([]kargoapi.PromotionRequestTarget, len(stageTargets))
	for i, target := range stageTargets {
		promotionRequestTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
	}

	return new(newPromotionRequest(stage, freightName, promotionRequestTargets, *requestUpdateStrategy)), nil
}

// NewPromotionRequestForSelectTargets creates a promotion request for specific subset of targets.
// If targets don't match the targets selected by the stage, ErrTargetNotSelected is returned.
func NewPromotionRequestForSelectTargets(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
	projectTargetsFn func(context.Context, string) ([]kargoapi.Target, error),
	updateStrategy *kargoapi.TargetUpdateStrategy,
	promotionRequestTargets []kargoapi.PromotionRequestTarget,
) (*kargoapi.PromotionRequest, error) {
	stageTargets, requestUpdateStrategy, err := precheckPromotionRequestFields(
		ctx, stage, projectTargetsFn, updateStrategy)
	if err != nil {
		return nil, err
	}

	// Validate that all requested targets are selected by the stage
	stageTargetsNameSet := make(map[string]any, len(stageTargets))
	for _, stageTarget := range stageTargets {
		stageTargetsNameSet[stageTarget.Name] = struct{}{}
	}
	for _, promotionRequestTarget := range promotionRequestTargets {
		if _, ok := stageTargetsNameSet[promotionRequestTarget.Name]; !ok {
			return nil, fmt.Errorf(
				"%w: target %q is not selected by stage %q",
				ErrTargetNotSelected,
				promotionRequestTarget.Name,
				stage.Name)
		}
	}

	return new(newPromotionRequest(stage, freightName, promotionRequestTargets, *requestUpdateStrategy)), nil
}

// precheckPromotionRequestFields initializes shared fields and runs shared checks
// for different NewPromotionRequest functions
func precheckPromotionRequestFields(
	ctx context.Context,
	stage *kargoapi.Stage,
	projectTargetsFn func(context.Context, string) ([]kargoapi.Target, error),
	updateStrategy *kargoapi.TargetUpdateStrategy,
) ([]kargoapi.Target, *kargoapi.PromotionRequestUpdateStrategy, error) {
	if stage == nil {
		return nil, nil, fmt.Errorf("cannot create promotion request without a stage")
	}
	if stage.Spec.Targets == nil {
		return nil, nil, fmt.Errorf("cannot create promotion request for a stage without targets")
	}

	stageTargets, err := getStageTargets(ctx, projectTargetsFn, stage)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create promotion request without stage targets: %w", err)
	}

	requestUpdateStrategy, err := ensureUpdateStrategy(updateStrategy, stage, stageTargets)
	if err != nil {
		return nil, nil, err
	}
	return stageTargets, requestUpdateStrategy, nil
}

// ensureUpdateStrategy calculates update strategy from stage targets if it's not set in updateStrategy
func ensureUpdateStrategy(
	updateStrategy *kargoapi.TargetUpdateStrategy,
	stage *kargoapi.Stage,
	stageTargets []kargoapi.Target,
) (*kargoapi.PromotionRequestUpdateStrategy, error) {
	if updateStrategy == nil {
		// Calculate effective update strategy for promotion request
		// This computes MaxConcurrentPercent and MaxFailuresPercent into MaxConcurrent and MaxFailures
		return calculateUpdateStrategy(stage.Spec.Targets.UpdateStrategy, len(stageTargets))
	}
	return calculateUpdateStrategy(*updateStrategy, len(stageTargets))
}

func getStageTargets(
	ctx context.Context,
	projectTargetsFn func(context.Context, string) ([]kargoapi.Target, error),
	stage *kargoapi.Stage,
) ([]kargoapi.Target, error) {
	targets, err := projectTargetsFn(ctx, stage.Namespace)
	if err != nil {
		return nil, err
	}
	targets, err = FilterTargetsForStage(stage, targets)
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// newPromotionRequest creates a promotion request for selected targets of the stage.
// updateStrategy is calculated for a stage, not selected targets.
func newPromotionRequest(
	stage *kargoapi.Stage,
	freightName string,
	promotionRequestTargets []kargoapi.PromotionRequestTarget,
	updateStrategy kargoapi.PromotionRequestUpdateStrategy,
) kargoapi.PromotionRequest {
	labels := map[string]string{kargoapi.LabelKeyStage: stage.Name}
	// The PromotionRequest reconciler filters by shard, so a PromotionRequest
	// must carry the shard of the Stage that it promotes on behalf of. Without
	// this, only the default controller would ever reconcile it.
	if stage.Spec.Shard != "" {
		labels[kargoapi.LabelKeyShard] = stage.Spec.Shard
	}

	return kargoapi.PromotionRequest{
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
			Stage:          stage.Name,
			Freight:        freightName,
			Targets:        promotionRequestTargets,
			UpdateStrategy: updateStrategy,
		},
	}
}

// calculateUpdateStrategy calculates effective updateStrategy for a PromotionRequest
// based on the stage updateStrategy and the number of targets selected.
// For updateStrategy used in PromotionRequest, the percentage values are not set.
func calculateUpdateStrategy(
	stageStrategy kargoapi.TargetUpdateStrategy,
	targetCount int,
) (*kargoapi.PromotionRequestUpdateStrategy, error) {
	maxConcurrent, err := calculateUpdateStrategyValue(stageStrategy.MaxConcurrent, targetCount)
	if err != nil {
		return nil, err
	}
	maxFailures, err := calculateUpdateStrategyValue(stageStrategy.MaxFailures, targetCount)
	if err != nil {
		return nil, err
	}
	return &kargoapi.PromotionRequestUpdateStrategy{
		MaxConcurrent: int64(maxConcurrent),
		MaxFailures:   int64(maxFailures),
	}, nil
}

func calculateUpdateStrategyValue(configValue *intstr.IntOrString, targetCount int) (int, error) {
	if configValue == nil {
		return 0, nil
	}
	return intstr.GetScaledValueFromIntOrPercent(configValue, targetCount, true)
}

// GetPromotionRequest returns a pointer to the PromotionRequest resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetPromotionRequest(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.PromotionRequest, error) {
	request := kargoapi.PromotionRequest{}
	if err := c.Get(ctx, namespacedName, &request); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting PromotionRequest %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &request, nil
}
