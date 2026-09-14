package projects

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
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

	targets := &kargoapi.TargetList{}
	if err := r.client.List(
		ctx,
		targets,
		client.InNamespace(project.Name),
	); err != nil {
		conditions.Set(&status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             "CollectingTargetStatsFailed",
			Message:            "Failed to collect Target stats: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return status, fmt.Errorf("error listing Targets: %w", err)
	}

	promotionRequests := &kargoapi.PromotionRequestList{}
	if err := r.client.List(
		ctx,
		promotionRequests,
		client.InNamespace(project.Name),
	); err != nil {
		conditions.Set(&status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             "CollectingTargetStatsFailed",
			Message:            "Failed to collect Target stats: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return status, fmt.Errorf("error listing PromotionRequests: %w", err)
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

	stats.Targets = collectTargetStats(
		targets.Items,
		stages.Items,
		promotionRequests.Items,
	)

	status.Stats = &stats

	conditions.Delete(&status, kargoapi.ConditionTypeHealthy)
	return status, nil
}

// collectTargetStats summarizes a Project's Targets and, across its
// target-aware Stages, the per-Target outcome of each Stage's latest
// PromotionRequest. It returns nil for a Project with no Targets and no
// target-aware Stages, so that classic Projects carry no fleet stats at all.
func collectTargetStats(
	targets []kargoapi.Target,
	stages []kargoapi.Stage,
	promotionRequests []kargoapi.PromotionRequest,
) *kargoapi.TargetStats {
	byName := make(map[string]*kargoapi.PromotionRequest, len(promotionRequests))
	for i := range promotionRequests {
		byName[promotionRequests[i].Name] = &promotionRequests[i]
	}

	stats := &kargoapi.TargetStats{Count: int64(len(targets))}
	targetAwareStages := 0
	for _, stage := range stages {
		if !api.IsTargetAware(&stage) {
			continue
		}
		targetAwareStages++
		// The latest round is the one the Stage reports as current, else as
		// last. A Stage that reports neither has never promoted and has no
		// round to account for.
		var name string
		if ref := stage.Status.CurrentPromotionRequest; ref != nil {
			name = ref.Name
		} else if ref := stage.Status.LastPromotionRequest; ref != nil {
			name = ref.Name
		}
		if name == "" {
			continue
		}
		request, ok := byName[name]
		if !ok {
			stats.Unknown++
			continue
		}
		addPromotionRequestSummary(&stats.Promotion, promotionSummary(request))
	}

	if stats.Count == 0 && targetAwareStages == 0 {
		return nil
	}
	return stats
}

// promotionSummary returns the per-Target outcome of a PromotionRequest. The
// fan-out controller records this in status.summary once it has created child
// Promotions. Before that, every Target the request names is pending; and a
// request that reached a terminal phase without ever fanning out -- because
// fan-out is unavailable, say -- ended every one of its Targets in that phase.
func promotionSummary(request *kargoapi.PromotionRequest) kargoapi.PromotionRequestSummary {
	if request.Status.Summary != nil {
		return *request.Status.Summary
	}
	count := int32(len(request.Spec.Targets)) // nolint: gosec
	switch request.Status.Phase {
	case kargoapi.PromotionRequestPhaseSucceeded:
		return kargoapi.PromotionRequestSummary{Succeeded: count}
	case kargoapi.PromotionRequestPhaseFailed:
		return kargoapi.PromotionRequestSummary{Failed: count}
	case kargoapi.PromotionRequestPhaseErrored:
		return kargoapi.PromotionRequestSummary{Errored: count}
	default:
		return kargoapi.PromotionRequestSummary{Pending: count}
	}
}

// addPromotionRequestSummary adds every count in src to dst.
func addPromotionRequestSummary(
	dst *kargoapi.PromotionRequestSummary,
	src kargoapi.PromotionRequestSummary,
) {
	dst.Pending += src.Pending
	dst.Running += src.Running
	dst.Succeeded += src.Succeeded
	dst.Failed += src.Failed
	dst.Errored += src.Errored
	dst.Aborted += src.Aborted
}
