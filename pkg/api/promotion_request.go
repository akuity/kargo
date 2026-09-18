package api

import (
	"context"
	"fmt"
	"strings"

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
// As with Promotions, the name is left to the defaulting webhook, which
// generates one that embeds the Freight so that sorting logic elsewhere in
// Kargo can read creation order from it.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRequest(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
	freightName string,
) (*kargoapi.PromotionRequest, error) {
	promotionRequest, err := newPromotionRequest(ctx, c, stage)
	if err != nil {
		return nil, err
	}
	promotionRequest.Spec.Freight = freightName
	return promotionRequest, nil
}

// NewPromotionRequestForOrigin constructs a PromotionRequest that, unlike
// NewPromotionRequest which is explicit about the exact Freight to promote,
// specifies only an origin. The PromotionRequest defaulting webhook resolves
// the origin to the auto-promotion candidate Freight at admission time.
//
// The Stage MUST be target-aware. Callers should gate on IsTargetAware.
func NewPromotionRequestForOrigin(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
	origin kargoapi.FreightOrigin,
) (*kargoapi.PromotionRequest, error) {
	promotionRequest, err := newPromotionRequest(ctx, c, stage)
	if err != nil {
		return nil, err
	}
	promotionRequest.Spec.Origin = &origin
	return promotionRequest, nil
}

// newPromotionRequest constructs a PromotionRequest with everything but the
// Freight (or the origin standing in for it).
func newPromotionRequest(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
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
			// The defaulting webhook overwrites this. We set it here only so that
			// the Kubernetes API server has a name to work with before admission
			// runs.
			GenerateName: "promoreq-",
			Labels:       labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					stage,
					kargoapi.GroupVersion.WithKind("Stage"),
				),
			},
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   stage.Name,
			Targets: specTargets,
		},
	}, nil
}
