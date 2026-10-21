package api

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

type PromotionRequestBuilder struct {
	stage                   *kargoapi.Stage
	freightName             string
	stageTargets            []kargoapi.Target
	promotionRequestTargets []kargoapi.PromotionRequestTarget
	updateStrategy          *kargoapi.PromotionRequestUpdateStrategy
}

func NewPromotionRequestBuilder(
	stage *kargoapi.Stage,
	freightName string,
) *PromotionRequestBuilder {
	return &PromotionRequestBuilder{
		stage:       stage,
		freightName: freightName,
	}
}

// WithStageTargets specifies the stageTargets for this request.
// Unless promotionRequestTargets is set, all these targets will be promoted to (default).
// If promotionRequestTargets is set, these targets are used
// to validate that promotionRequestTargets are selected by the stage.
// This targets list is also used to calculate effective update strategy unless custom updateStrategy is set
func (builder *PromotionRequestBuilder) WithStageTargets(
	stageTargets []kargoapi.Target,
) *PromotionRequestBuilder {
	builder.stageTargets = stageTargets
	return builder
}

// WithPromotionRequestTargets overrides targets to be promoted to.
// If set, only those targets will be promoted to, instead of all the stageTargets.
// These targets need to be a subset of stageTargets.
func (builder *PromotionRequestBuilder) WithPromotionRequestTargets(
	promotionRequestTargets []kargoapi.PromotionRequestTarget,
) *PromotionRequestBuilder {
	builder.promotionRequestTargets = promotionRequestTargets
	return builder
}

// WithUpdateStrategy overrides updateStrategy for the request.
// If not set, the updateStrategy is calculated from the stages updateStrategy and a number of stageTargets.
// If set, the configured updateStrategy is used regardless of the stage configuration.
func (builder *PromotionRequestBuilder) WithUpdateStrategy(
	updateStrategy *kargoapi.PromotionRequestUpdateStrategy,
) *PromotionRequestBuilder {
	builder.updateStrategy = updateStrategy
	return builder
}

// Build constructs a new PromotionRequest object without creating it in the API.
// It calculates targets and updateStrategy based on builder configuration.
func (builder *PromotionRequestBuilder) Build(
	ctx context.Context,
	c client.Client,
) (*kargoapi.PromotionRequest, error) {
	if builder.stage == nil {
		return nil, fmt.Errorf("cannot create promotion request without a stage")
	}
	if builder.stage.Spec.Targets == nil {
		return nil, fmt.Errorf("cannot create promotion request for a stage without targets")
	}

	stageTargets := builder.stageTargets
	var err error

	if stageTargets == nil {
		stageTargets, err = ListTargetsForStage(ctx, c, builder.stage)
		if err != nil {
			return nil, fmt.Errorf(
				"error resolving Targets governed by Stage %q in namespace %q: %w",
				builder.stage.Name, builder.stage.Namespace, err,
			)
		}
	}

	updateStrategy := builder.updateStrategy
	if updateStrategy == nil {
		// Calculate effective update strategy for promotion request
		// This computes MaxConcurrentPercent and MaxFailuresPercent into MaxConcurrent and MaxFailures
		updateStrategy, err = calculateUpdateStrategy(builder.stage.Spec.Targets.UpdateStrategy, len(stageTargets))
		if err != nil {
			return nil, err
		}
	}

	promotionRequestTargets := builder.promotionRequestTargets

	// If the builder did not supply the list of targets, we assume all stage targets need to be added
	if promotionRequestTargets == nil {
		// Never nil: spec.targets is a required field, so a nil slice would
		// serialize as null and be rejected. An empty list is meaningful -- it
		// records that the Stage governed no Targets at this moment.
		promotionRequestTargets = make([]kargoapi.PromotionRequestTarget, len(stageTargets))
		for i, target := range stageTargets {
			promotionRequestTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
		}
	} else {
		// Validate that all requested targets are selected by the stage
		stageTargetsNameSet := make(map[string]any, len(stageTargets))
		for _, stageTarget := range stageTargets {
			stageTargetsNameSet[stageTarget.Name] = struct{}{}
		}
		for _, promotionRequestTarget := range promotionRequestTargets {
			if _, ok := stageTargetsNameSet[promotionRequestTarget.Name]; !ok {
				return nil, fmt.Errorf(
					"target %q is not selected by stage %q",
					promotionRequestTarget.Name,
					builder.stage.Name)
			}
		}
	}

	return newPromotionRequest(builder.stage, builder.freightName, promotionRequestTargets, *updateStrategy)

}

// newPromotionRequest creates a promotion request for selected targets of the stage.
// updateStrategy is calculated for a stage, not selected targets.
func newPromotionRequest(
	stage *kargoapi.Stage,
	freightName string,
	promotionRequestTargets []kargoapi.PromotionRequestTarget,
	updateStrategy kargoapi.PromotionRequestUpdateStrategy,
) (*kargoapi.PromotionRequest, error) {
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
			Stage:          stage.Name,
			Freight:        freightName,
			Targets:        promotionRequestTargets,
			UpdateStrategy: updateStrategy,
		},
	}, nil
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
