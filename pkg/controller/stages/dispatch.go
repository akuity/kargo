package stages

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	argocdapi "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion/dispatch"
)

const (
	// maxDispatchCandidates caps how many Pending Promotions are evaluated
	// per pass. A Promotion held at the head of the queue must not starve a
	// permitted one behind it (e.g. a rollback during a freeze), so
	// candidates are evaluated in queue order until one is allowed.
	maxDispatchCandidates = 10

	// Bounds on the requeue interval when dispatch is blocked. The lower
	// bound guards against hot-looping on a policy that returns a tiny (or
	// zero) requeue; the upper bound ensures a Stage re-evaluates its held
	// Promotions periodically even if the policy's "when" is far away.
	minDispatchRequeue     = 5 * time.Second
	maxDispatchRequeue     = 30 * time.Minute
	defaultDispatchRequeue = time.Minute
)

// Event reasons emitted by the dispatch gate.
const (
	eventReasonPromotionBlocked    = "PromotionBlocked"
	eventReasonPromotionPolicyErr  = "PromotionPolicyError"
	conditionReasonDispatchBlocked = "DispatchBlocked"
)

// Annotation keys carrying the structured dispatch hold reason on a
// PromotionBlocked event, so consumers need not parse the free-text message.
const (
	annotationKeyDispatchRules     = "kargo.akuity.io/dispatch-rules"
	annotationKeyDispatchBlockedBy = "kargo.akuity.io/dispatch-blocked-by"
	annotationKeyDispatchUntil     = "kargo.akuity.io/dispatch-until"
)

// conditionReasonForRule maps a dispatch violation rule to a CamelCase Stage
// condition Reason token. Unmapped rules (e.g. those from a custom policy)
// yield no token, so the condition falls back to the generic
// conditionReasonDispatchBlocked.
var conditionReasonForRule = map[string]string{
	"windows":           "OutsideWindow",
	"freezes":           "Frozen",
	"ratelimit":         "RateLimited",
	"yield-to-rollback": "YieldToRollback",
	"yield-to-manual":   "YieldToManual",
	"regression":        "Regression",
	"would-regress":     "WouldRegress",
	"auto-hold":         "AutoHeld",
	"scheduled":         "Scheduled",
}

// dispatchEventAnnotations projects a held decision's structured reasons into
// event annotations: the distinct rules, any Promotions deferred to, and the
// soonest self-clear time. Returns nil when there is nothing structured to add.
func dispatchEventAnnotations(reasons []dispatch.Reason) map[string]string {
	var rules, blockedBy []string
	seen := map[string]bool{}
	var until *time.Time
	for _, r := range reasons {
		if r.Rule != "" && !seen[r.Rule] {
			seen[r.Rule] = true
			rules = append(rules, r.Rule)
		}
		if r.BlockedBy != "" {
			blockedBy = append(blockedBy, r.BlockedBy)
		}
		if r.Until != nil && (until == nil || r.Until.Before(*until)) {
			until = r.Until
		}
	}
	ann := map[string]string{}
	if len(rules) > 0 {
		ann[annotationKeyDispatchRules] = strings.Join(rules, ",")
	}
	if len(blockedBy) > 0 {
		ann[annotationKeyDispatchBlockedBy] = strings.Join(blockedBy, ",")
	}
	if until != nil {
		ann[annotationKeyDispatchUntil] = until.UTC().Format(time.RFC3339)
	}
	if len(ann) == 0 {
		return nil
	}
	return ann
}

// conditionReasonForHeld derives the Stage Promoting-condition Reason from the
// set of rules that held the evaluated candidates: the rule's token when a
// single mapped rule is responsible, else the generic blocked reason.
func conditionReasonForHeld(rules map[string]struct{}) string {
	if len(rules) == 1 {
		for rule := range rules {
			if token := conditionReasonForRule[rule]; token != "" {
				return token
			}
		}
	}
	return conditionReasonDispatchBlocked
}

// isPendingPhase returns whether the Promotion phase counts as awaiting
// dispatch. A brand-new Promotion has an empty phase until the promotion
// reconciler marks it Pending; both must be gated or a Promotion could slip
// through the gate in that window.
func isPendingPhase(phase kargoapi.PromotionPhase) bool {
	return phase == kargoapi.PromotionPhasePending || phase == ""
}

// gateDispatch is the promotion dispatch gate. Given the Stage's Promotions
// (sorted by api.ComparePromotionByPhaseAndCreationTime), it evaluates the
// dispatch policy against each Pending Promotion in queue order and returns
// the first one the policy allows. When every candidate is held, it returns
// a nil Promotion along with how long to wait before re-evaluating, a
// human-readable message, and a Stage condition Reason token summarizing why.
//
// Policy errors fail closed: a Stage with a broken policy does not dispatch
// until the policy is fixed. The error is surfaced on the Promotion as an
// event and to the caller.
func (r *RegularStageReconciler) gateDispatch(
	ctx context.Context,
	stage *kargoapi.Stage,
	promos []kargoapi.Promotion,
) (*kargoapi.Promotion, time.Duration, string, string, error) {
	logger := logging.LoggerFromContext(ctx)

	head := firstPending(promos)
	if head == nil {
		// Nothing awaiting dispatch; nothing to gate.
		return nil, 0, "", "", nil
	}

	// Gather policy configuration. When there is none, the gate is a no-op
	// and the head of the queue dispatches, exactly as before this gate
	// existed.
	projectCfg := &kargoapi.ProjectConfig{}
	if err := r.client.Get(
		ctx,
		types.NamespacedName{Namespace: stage.Namespace, Name: stage.Namespace},
		projectCfg,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, 0, "", "", fmt.Errorf(
				"error getting ProjectConfig for Project %q: %w", stage.Namespace, err,
			)
		}
		projectCfg = nil
	}
	clusterCfg, err := api.GetClusterConfig(ctx, r.client)
	if err != nil {
		return nil, 0, "", "", err
	}
	var projectSpec *kargoapi.ProjectConfigSpec
	if projectCfg != nil {
		projectSpec = &projectCfg.Spec
	}
	var freezes []kargoapi.PromotionFreeze
	var clusterCustom string
	if clusterCfg != nil {
		freezes = clusterCfg.Spec.PromotionFreezes
		clusterCustom = clusterCfg.Spec.CustomPolicy
	}
	var projectCustom string
	governed := len(freezes) > 0 || clusterCustom != ""
	if projectSpec != nil {
		projectCustom = projectSpec.CustomPolicy
		governed = governed ||
			projectCustom != "" ||
			len(projectSpec.PromotionWindows) > 0 ||
			len(projectSpec.RateLimits) > 0
	}
	if !governed {
		return head, 0, "", "", nil
	}

	now := time.Now()

	// Times at which this Stage's Promotions were dispatched (began
	// Running), for the rate-limit block's rolling window, and the queue of
	// Promotions still awaiting dispatch, for policies that reason about the
	// backlog. Both are derived from the Promotions already listed by the
	// caller (in gate order); no extra state is kept. The queue is not capped
	// at maxDispatchCandidates: the cap bounds evaluation, not visibility, so
	// a policy can gauge true backlog depth.
	var dispatches []time.Time
	var queue []kargoapi.Promotion
	for i := range promos {
		if startedAt := promos[i].Status.StartedAt; startedAt != nil {
			dispatches = append(dispatches, startedAt.Time)
		}
		if isPendingPhase(promos[i].Status.Phase) {
			queue = append(queue, promos[i])
		}
	}

	// Project metadata gives policies a lightweight way to be data-driven
	// (including project-scoped freezes); tolerate its absence.
	project, err := api.GetProject(ctx, r.client, stage.Namespace)
	if err != nil {
		logger.Error(err, "error getting Project for dispatch policy input")
	}

	// The Stage's current Freight per origin, so a policy can tell whether a
	// candidate advances or regresses the Stage (data.currentFreight). A
	// genuine fetch error fails closed, like the candidate Freight below.
	currentFreight, err := r.resolveCurrentFreight(ctx, stage)
	if err != nil {
		return nil, 0, "", "", err
	}

	// The Stage's committed auto-promotion holds per origin, so the gate can
	// deny an auto-forward for a held origin (data.autoPromotionHolds) — the
	// dispatch-side complement of the controller's creation-side hold check.
	// Read directly off the in-memory Stage; no fetch needed.
	data, err := dispatch.BuildData(
		projectSpec, freezes, stage, project, dispatches, queue, currentFreight,
		stage.Status.AutoPromotionHolds,
	)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("error building dispatch policy data: %w", err)
	}

	// The Argo CD Applications this Stage is authorized to manage, for
	// server-scoped freezes. Empty when the Argo CD integration is
	// disabled, in which case server-scoped freezes never match.
	var apps []argocdapi.Application
	if r.argocdClient != nil {
		appList := &argocdapi.ApplicationList{}
		if err = r.argocdClient.List(ctx, appList, client.MatchingFields{
			indexer.ApplicationsByAuthorizedStageField: stage.Namespace + ":" + stage.Name,
		}); err != nil {
			return nil, 0, "", "", fmt.Errorf("error listing Argo CD Applications for Stage: %w", err)
		}
		apps = appList.Items
	}

	var msgs []string
	var minRequeue time.Duration
	heldRules := map[string]struct{}{}
	evaluated := 0
	for i := range promos {
		promo := &promos[i]
		if !isPendingPhase(promo.Status.Phase) || evaluated >= maxDispatchCandidates {
			continue
		}
		evaluated++

		freight, err := api.GetFreight(ctx, r.client, types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Freight,
		})
		if err != nil {
			return nil, 0, "", "", fmt.Errorf(
				"error getting Freight %q for Promotion %q: %w",
				promo.Spec.Freight, promo.Name, err,
			)
		}

		input := dispatch.BuildInput(promo, freight, stage, project, apps, now)
		decision, err := r.dispatchEngine.Evaluate(ctx, projectCustom, clusterCustom, input, data)
		if err != nil {
			r.recorder.Eventf(
				promo, corev1.EventTypeWarning, eventReasonPromotionPolicyErr,
				"dispatch policy failed; promotion will not be dispatched until the policy is fixed: %s",
				err.Error(),
			)
			return nil, 0, "", "", fmt.Errorf(
				"error evaluating dispatch policy for Promotion %q: %w", promo.Name, err,
			)
		}
		if decision.Allow {
			logger.Debug(
				"dispatch policy allowed Promotion",
				"promotion", promo.Name,
				"message", decision.Message,
			)
			return promo, 0, "", "", nil
		}

		logger.Debug(
			"dispatch policy held Promotion",
			"promotion", promo.Name,
			"message", decision.Message,
			"requeueAfter", decision.RequeueAfter,
		)
		// Deny messages are designed to be stable while the denial lasts, so
		// the recorder aggregates repeats instead of spamming events. The
		// structured reason rides along as event annotations so consumers need
		// not parse the message.
		r.recorder.AnnotatedEventf(
			promo, dispatchEventAnnotations(decision.Reasons),
			corev1.EventTypeNormal, eventReasonPromotionBlocked,
			"%s", decision.Message,
		)
		for _, reason := range decision.Reasons {
			if reason.Rule != "" {
				heldRules[reason.Rule] = struct{}{}
			}
		}
		msgs = append(msgs, fmt.Sprintf("Promotion %q: %s", promo.Name, decision.Message))
		if decision.RequeueAfter > 0 &&
			(minRequeue == 0 || decision.RequeueAfter < minRequeue) {
			minRequeue = decision.RequeueAfter
		}
	}

	blockedFor := minRequeue
	if blockedFor == 0 {
		blockedFor = defaultDispatchRequeue
	}
	blockedFor = min(max(blockedFor, minDispatchRequeue), maxDispatchRequeue)
	return nil, blockedFor, strings.Join(msgs, "; "), conditionReasonForHeld(heldRules), nil
}

// firstPending returns the first Promotion awaiting dispatch, or nil.
func firstPending(promos []kargoapi.Promotion) *kargoapi.Promotion {
	for i := range promos {
		if isPendingPhase(promos[i].Status.Phase) {
			return &promos[i]
		}
	}
	return nil
}

// resolveCurrentFreight projects the Stage's current Freight, per origin, into
// the dispatch policy's data.currentFreight. The resolution (fetching each
// current Freight to recover its discovery time, omitting garbage-collected
// origins, failing closed on any other error) is shared with the Promotion
// webhook via api.GetCurrentFreight, so the gate and the webhook agree on the
// current Freight for an origin.
func (r *RegularStageReconciler) resolveCurrentFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
) (map[string]dispatch.CurrentFreight, error) {
	current, err := api.GetCurrentFreight(ctx, r.client, stage)
	if err != nil {
		return nil, err
	}
	resolved := make(map[string]dispatch.CurrentFreight, len(current))
	for origin, freight := range current {
		resolved[origin] = dispatch.CurrentFreight{
			Name:         freight.Name,
			DiscoveredAt: freight.EffectiveDiscoveredAt(),
		}
	}
	return resolved, nil
}

// enqueueStagesForConfigChange returns reconcile requests for every regular
// Stage this reconciler is responsible for, optionally narrowed to one
// namespace (Project). Used to promptly re-evaluate held Promotions when
// dispatch policy configuration changes.
func (r *RegularStageReconciler) enqueueStagesForConfigChange(
	ctx context.Context,
	namespace string,
) []reconcile.Request {
	stages := &kargoapi.StageList{}
	var opts []client.ListOption
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := r.client.List(ctx, stages, opts...); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "error listing Stages for dispatch policy config change",
		)
		return nil
	}
	var reqs []reconcile.Request
	for i := range stages.Items {
		stage := &stages.Items[i]
		if stage.IsControlFlow() || !r.shardPredicate.IsResponsible(stage) {
			continue
		}
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: stage.Namespace,
				Name:      stage.Name,
			},
		})
	}
	return reqs
}
