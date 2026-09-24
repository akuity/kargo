package projects

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
)

// collectStats collects statistics about the current health state of all
// Warehouses and Stages in the Project, and about its Targets and the latest
// round of promotion to them. It returns a ProjectStatus that's been updated
// with the collected stats.
func (r *reconciler) collectStats(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	if cond := conditions.Get(
		&project.Status,
		kargoapi.ConditionTypeReady,
	); cond == nil || cond.Status != metav1.ConditionTrue {
		logging.LoggerFromContext(ctx).Debug("Project is not ready; won't collect stats")
		return project.Status, nil
	}

	status := *project.Status.DeepCopy()

	warehouses := &kargoapi.WarehouseList{}
	if err := r.client.List(
		ctx,
		warehouses,
		client.InNamespace(project.Name),
	); err != nil {
		conditions.Set(&status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             "CollectingWarehouseStatsFailed",
			Message:            "Failed to collect Warehouse stats: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return status, fmt.Errorf("error listing Warehouses: %w", err)
	}

	stages := &kargoapi.StageList{}
	if err := r.client.List(
		ctx,
		stages,
		client.InNamespace(project.Name),
	); err != nil {
		conditions.Set(&status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             "CollectingStageStatsFailed",
			Message:            "Failed to collect Stage stats: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return status, fmt.Errorf("error listing Stages: %w", err)
	}

	// Targets and PromotionRequests live in the database. Without one, the
	// Project has no Target stats.
	var targets []kargoapi.Target
	var promotionRequests []kargoapi.PromotionRequest
	if r.store != nil {
		targetStatsFailed := func(what string, err error) {
			conditions.Set(&status, &metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionFalse,
				Reason:             "CollectingTargetStatsFailed",
				Message:            "Failed to list " + what + ": " + err.Error(),
				ObservedGeneration: project.GetGeneration(),
			})
		}
		targetRows, err := r.store.ListTargets(ctx, project.Name)
		if err != nil {
			targetStatsFailed("targets", err)
			return status, fmt.Errorf("error listing Targets: %w", err)
		}
		if targets, err = database.TargetsFromRows(targetRows, project.Name); err != nil {
			targetStatsFailed("targets", err)
			return status, fmt.Errorf("error listing Targets: %w", err)
		}
		snapshots, err := r.store.ListPromotionRequests(ctx, project.Name)
		if err != nil {
			targetStatsFailed("promotion requests", err)
			return status, fmt.Errorf("error listing PromotionRequests: %w", err)
		}
		promotionRequests = database.PromotionRequestsFromSnapshots(snapshots)
	}

	controlFlowStages := 0
	for _, stage := range stages.Items {
		if stage.IsControlFlow() {
			controlFlowStages++
		}
	}

	stats := kargoapi.ProjectStats{
		Warehouses: kargoapi.WarehouseStats{
			Count: int64(len(warehouses.Items)),
		},
		Stages: kargoapi.StageStats{
			Count: int64(len(stages.Items)) - int64(controlFlowStages),
		},
	}

	for _, warehouse := range warehouses.Items {
		if health := conditions.Get(
			&warehouse.Status,
			kargoapi.ConditionTypeHealthy,
		); health != nil && health.Status == metav1.ConditionTrue {
			stats.Warehouses.Health.Healthy++
		}
	}

	for _, stage := range stages.Items {
		if health := conditions.Get(
			&stage.Status,
			kargoapi.ConditionTypeHealthy,
		); health != nil && health.Status == metav1.ConditionTrue {
			stats.Stages.Health.Healthy++
		}
	}

	stats.Targets = collectTargetStats(targets, stages.Items, promotionRequests)

	status.Stats = &stats

	conditions.Delete(&status, kargoapi.ConditionTypeHealthy)
	return status, nil
}

// collectTargetStats summarizes a Project's Targets: how many there are, and
// how many are healthy. It returns nil for a Project with no Targets, so that
// classic Projects carry no Target stats at all.
//
// Target status does not yet record health, so a Target is counted as healthy
// when at least one Stage has promoted to it and the latest promotion to it
// from every Stage that did so succeeded. The latest promotion from a Stage is
// the child Promotion for the Target in the PromotionRequest the Stage reports
// as current, else as last. A Stage whose latest request no longer exists says
// nothing about its Targets and is skipped.
func collectTargetStats(
	targets []kargoapi.Target,
	stages []kargoapi.Stage,
	promotionRequests []kargoapi.PromotionRequest,
) *kargoapi.TargetStats {
	if len(targets) == 0 {
		return nil
	}

	requestsByName := make(map[string]*kargoapi.PromotionRequest, len(promotionRequests))
	for i := range promotionRequests {
		requestsByName[promotionRequests[i].Name] = &promotionRequests[i]
	}

	// succeeded records, for every Target named by some Stage's latest
	// request, whether every such request's promotion to it succeeded.
	succeeded := make(map[string]bool, len(targets))
	for i := range stages {
		request := latestPromotionRequest(&stages[i], requestsByName)
		if request == nil {
			continue
		}
		for name, phase := range targetPhases(request) {
			ok, seen := succeeded[name]
			succeeded[name] = (!seen || ok) && phase == kargoapi.PromotionPhaseSucceeded
		}
	}

	stats := &kargoapi.TargetStats{Count: int64(len(targets))}
	for _, target := range targets {
		if succeeded[target.Name] {
			stats.Health.Healthy++
		}
	}
	return stats
}

// latestPromotionRequest returns the PromotionRequest a target-aware Stage
// reports as current, else as last, or nil when the Stage is classic, has
// never promoted, or reports a request that no longer exists.
func latestPromotionRequest(
	stage *kargoapi.Stage,
	requestsByName map[string]*kargoapi.PromotionRequest,
) *kargoapi.PromotionRequest {
	if !api.IsTargetAware(stage) {
		return nil
	}
	var name string
	if ref := stage.Status.CurrentPromotionRequest; ref != nil {
		name = ref.Name
	} else if ref := stage.Status.LastPromotionRequest; ref != nil {
		name = ref.Name
	}
	return requestsByName[name]
}

// targetPhases returns the phase of the promotion to each Target a
// PromotionRequest names. The fan-out controller records a phase per Target
// in status.targets once it has created that Target's child Promotion. A
// Target without one is pending while the request runs, and shares the
// request's own phase once it has ended -- a request that ended without ever
// fanning out ended every one of its Targets that way.
func targetPhases(request *kargoapi.PromotionRequest) map[string]kargoapi.PromotionPhase {
	var fallback kargoapi.PromotionPhase
	switch request.Status.Phase {
	case kargoapi.PromotionRequestPhaseSucceeded:
		fallback = kargoapi.PromotionPhaseSucceeded
	case kargoapi.PromotionRequestPhaseFailed:
		fallback = kargoapi.PromotionPhaseFailed
	case kargoapi.PromotionRequestPhaseErrored:
		fallback = kargoapi.PromotionPhaseErrored
	default:
		fallback = kargoapi.PromotionPhasePending
	}
	phases := make(map[string]kargoapi.PromotionPhase, len(request.Spec.Targets))
	for _, target := range request.Spec.Targets {
		phases[target.Name] = fallback
	}
	for _, target := range request.Status.Targets {
		if target.Phase != "" {
			phases[target.Name] = target.Phase
		}
	}
	return phases
}
