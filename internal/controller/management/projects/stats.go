package projects

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
)

func (r *reconciler) collectStats(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	// Mark the Project as reconciling.
	conditions.Set(&project.Status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "CollectingProjectStats",
		Message:            "Collecting project stats",
		ObservedGeneration: project.GetGeneration(),
	})

	status := *project.Status.DeepCopy()

	warehouses := &kargoapi.WarehouseList{}
	if err := r.client.List(
		ctx,
		warehouses,
		client.InNamespace(project.Name),
	); err != nil {
		conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
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
		conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
		conditions.Set(&status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             "CollectingStageStatsFailed",
			Message:            "Failed to collect Stage stats: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return status, fmt.Errorf("error listing Stages: %w", err)
	}

	stats := kargoapi.ProjectStats{}

	for _, warehouse := range warehouses.Items {
		health := conditions.Get(&warehouse.Status, kargoapi.ConditionTypeHealthy)
		if health == nil {
			stats.Warehouses.Health.Unknown++
			continue
		}
		switch health.Status {
		case metav1.ConditionTrue:
			stats.Warehouses.Health.Healthy++
		case metav1.ConditionFalse:
			stats.Warehouses.Health.Unhealthy++
		default:
			stats.Warehouses.Health.Unknown++
		}
	}

	for _, stage := range stages.Items {
		health := conditions.Get(&stage.Status, kargoapi.ConditionTypeHealthy)
		if health == nil {
			stats.Stages.Health.Unknown++
			continue
		}
		switch health.Status {
		case metav1.ConditionTrue:
			stats.Stages.Health.Healthy++
		case metav1.ConditionFalse:
			stats.Stages.Health.Unhealthy++
		default:
			stats.Stages.Health.Unknown++
		}
	}

	status.Stats = stats

	conditions.Delete(&status, kargoapi.ConditionTypeHealthy)
	conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
	return status, nil
}
