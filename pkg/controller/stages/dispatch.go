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
// a nil Promotion along with how long to wait before re-evaluating and a
// human-readable reason.
//
// Policy errors fail closed: a Stage with a broken policy does not dispatch
// until the policy is fixed. The error is surfaced on the Promotion as an
// event and to the caller.
func (r *RegularStageReconciler) gateDispatch(
	ctx context.Context,
	stage *kargoapi.Stage,
	promos []kargoapi.Promotion,
) (*kargoapi.Promotion, time.Duration, string, error) {
	logger := logging.LoggerFromContext(ctx)

	head := firstPending(promos)
	if head == nil {
		// Nothing awaiting dispatch; nothing to gate.
		return nil, 0, "", nil
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
			return nil, 0, "", fmt.Errorf(
				"error getting ProjectConfig for Project %q: %w", stage.Namespace, err,
			)
		}
		projectCfg = nil
	}
	clusterCfg, err := api.GetClusterConfig(ctx, r.client)
	if err != nil {
		return nil, 0, "", err
	}
	var projectSpec *kargoapi.ProjectConfigSpec
	if projectCfg != nil {
		projectSpec = &projectCfg.Spec
	}
	var exclusions []kargoapi.PromotionExclusion
	var clusterCustom string
	if clusterCfg != nil {
		exclusions = clusterCfg.Spec.PromotionExclusions
		clusterCustom = clusterCfg.Spec.CustomPolicy
	}
	var projectCustom string
	governed := len(exclusions) > 0 || clusterCustom != ""
	if projectSpec != nil {
		projectCustom = projectSpec.CustomPolicy
		governed = governed ||
			projectCustom != "" ||
			len(projectSpec.PromotionWindows) > 0 ||
			len(projectSpec.RateLimits) > 0
	}
	if !governed {
		return head, 0, "", nil
	}

	now := time.Now()

	// Times at which this Stage's Promotions were dispatched (began
	// Running), for the rate-limit block's rolling window. Derived from the
	// Promotions already listed by the caller; no extra state is kept.
	var dispatches []time.Time
	for i := range promos {
		if startedAt := promos[i].Status.StartedAt; startedAt != nil {
			dispatches = append(dispatches, startedAt.Time)
		}
	}

	// Project metadata gives policies a lightweight way to be data-driven
	// (including project-scoped exclusions); tolerate its absence.
	project, err := api.GetProject(ctx, r.client, stage.Namespace)
	if err != nil {
		logger.Error(err, "error getting Project for dispatch policy input")
	}

	data, err := dispatch.BuildData(projectSpec, exclusions, stage, project, dispatches)
	if err != nil {
		return nil, 0, "", fmt.Errorf("error building dispatch policy data: %w", err)
	}

	// The Argo CD Applications this Stage is authorized to manage, for
	// server-scoped exclusions. Empty when the Argo CD integration is
	// disabled, in which case server-scoped exclusions never match.
	var apps []argocdapi.Application
	if r.argocdClient != nil {
		appList := &argocdapi.ApplicationList{}
		if err = r.argocdClient.List(ctx, appList, client.MatchingFields{
			indexer.ApplicationsByAuthorizedStageField: stage.Namespace + ":" + stage.Name,
		}); err != nil {
			return nil, 0, "", fmt.Errorf("error listing Argo CD Applications for Stage: %w", err)
		}
		apps = appList.Items
	}

	var msgs []string
	var minRequeue time.Duration
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
			return nil, 0, "", fmt.Errorf(
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
			return nil, 0, "", fmt.Errorf(
				"error evaluating dispatch policy for Promotion %q: %w", promo.Name, err,
			)
		}
		if decision.Allow {
			logger.Debug(
				"dispatch policy allowed Promotion",
				"promotion", promo.Name,
				"message", decision.Message,
			)
			return promo, 0, "", nil
		}

		logger.Debug(
			"dispatch policy held Promotion",
			"promotion", promo.Name,
			"message", decision.Message,
			"requeueAfter", decision.RequeueAfter,
		)
		// Deny messages are designed to be stable while the denial lasts, so
		// the recorder aggregates repeats instead of spamming events.
		r.recorder.Eventf(
			promo, corev1.EventTypeNormal, eventReasonPromotionBlocked,
			"%s", decision.Message,
		)
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
	return nil, blockedFor, strings.Join(msgs, "; "), nil
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
