package projects

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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

	// Targets live in the database, as do the PromotionRollouts that promote
	// to them and the record of their Promotions, so the database counts
	// which Targets are healthy. Without one there are none to count.
	if r.targets != nil {
		counts, err := r.targets.Count(ctx, project.Name)
		if err != nil {
			conditions.Set(&status, &metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionFalse,
				Reason:             "CollectingTargetStatsFailed",
				Message:            "Failed to count targets: " + err.Error(),
				ObservedGeneration: project.GetGeneration(),
			})
			return status, fmt.Errorf("error counting Targets: %w", err)
		}
		stats.Targets = targetStats(counts)
	}

	status.Stats = &stats

	conditions.Delete(&status, kargoapi.ConditionTypeHealthy)
	return status, nil
}

// targetStats summarizes a Project's Targets: how many there are, and how
// many are healthy. It returns nil for a Project with no Targets, so that
// classic Projects carry no Target stats at all.
func targetStats(counts database.CountTargetsRow) *kargoapi.TargetStats {
	if counts.Total == 0 {
		return nil
	}
	return &kargoapi.TargetStats{
		Count:  counts.Total,
		Health: kargoapi.HealthStats{Healthy: counts.Healthy},
	}
}
