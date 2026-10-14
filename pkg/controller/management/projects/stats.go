package projects

import (
	"context"
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
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

	// Targets live in the database. Without one there are none to count.
	if r.targets != nil {
		rollouts, err := r.latestRollouts(ctx, project.Name, stages.Items)
		if err != nil {
			conditions.Set(&status, &metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionFalse,
				Reason:             "CollectingTargetStatsFailed",
				Message:            "Failed to read promotion rollouts: " + err.Error(),
				ObservedGeneration: project.GetGeneration(),
			})
			return status, err
		}
		counts, err := r.targets.Count(
			ctx,
			project.Name,
			healthyTargetNames(stages.Items, rollouts),
		)
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
		Health: kargoapi.HealthStats{Healthy: counts.Among},
	}
}

// latestRollouts reads the PromotionRollouts the target-aware Stages report
// as current or last, and what has become of each of their Targets, which
// the Promotions of the Project say.
func (r *reconciler) latestRollouts(
	ctx context.Context,
	project string,
	stages []kargoapi.Stage,
) ([]kargoapi.PromotionRollout, error) {
	names := latestRolloutNames(stages)
	if len(names) == 0 {
		return nil, nil
	}
	stored, err := r.rollouts.GetMany(ctx, project, names)
	if err != nil {
		return nil, fmt.Errorf("error reading PromotionRollouts: %w", err)
	}
	promos := &kargoapi.PromotionList{}
	if err = r.client.List(ctx, promos, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("error listing Promotions: %w", err)
	}
	rollouts := make([]kargoapi.PromotionRollout, len(stored))
	for i, s := range stored {
		rollout := promotionrolloutstore.ToAPI(s)
		rollout.Status.Targets, rollout.Status.Summary = api.PromotionRolloutTargetStatuses(
			rollout.Name,
			promos.Items,
		)
		rollouts[i] = *rollout
	}
	return rollouts, nil
}

// latestRolloutNames returns, sorted and without duplicates, the names of
// the PromotionRollouts the target-aware Stages report as current, else as
// last.
func latestRolloutNames(stages []kargoapi.Stage) []string {
	var names []string
	for i := range stages {
		if name := latestRolloutName(&stages[i]); name != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// healthyTargetNames returns, in order, the names of the Targets that count
// as healthy. Target status does not yet record health, so a Target is
// counted as healthy when at least one Stage has promoted to it and the
// latest promotion to it from every Stage that did so succeeded. The latest
// promotion from a Stage is the child Promotion for the Target in the
// PromotionRollout the Stage reports as current, else as last. A Stage whose
// latest rollout no longer exists says nothing about its Targets and is
// skipped.
func healthyTargetNames(
	stages []kargoapi.Stage,
	rollouts []kargoapi.PromotionRollout,
) []string {
	rolloutsByName := make(map[string]*kargoapi.PromotionRollout, len(rollouts))
	for i := range rollouts {
		rolloutsByName[rollouts[i].Name] = &rollouts[i]
	}

	// succeeded records, for every Target named by some Stage's latest
	// rollout, whether every such rollout's promotion to it succeeded.
	succeeded := map[string]bool{}
	for i := range stages {
		rollout := rolloutsByName[latestRolloutName(&stages[i])]
		if rollout == nil {
			continue
		}
		for name, phase := range targetPhases(rollout) {
			ok, seen := succeeded[name]
			succeeded[name] = (!seen || ok) && phase == kargoapi.PromotionPhaseSucceeded
		}
	}

	names := make([]string, 0, len(succeeded))
	for name, ok := range succeeded {
		if ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// latestRolloutName returns the name of the PromotionRollout a target-aware
// Stage reports as current, else as last, or an empty string when the Stage
// is classic or has never promoted.
func latestRolloutName(stage *kargoapi.Stage) string {
	if !stage.IsTargetAware() {
		return ""
	}
	if ref := stage.Status.CurrentPromotionRequest; ref != nil {
		return ref.Name
	}
	if ref := stage.Status.LastPromotionRequest; ref != nil {
		return ref.Name
	}
	return ""
}

// targetPhases returns the phase of the promotion to each Target a
// PromotionRollout names. status.targets records a phase per Target once
// that Target's child Promotion exists. A Target without one is pending
// while the rollout runs, and shares the rollout's own phase once it has
// ended -- a rollout that ended without ever fanning out ended every one of
// its Targets that way.
func targetPhases(rollout *kargoapi.PromotionRollout) map[string]kargoapi.PromotionPhase {
	var fallback kargoapi.PromotionPhase
	switch rollout.Status.Phase {
	case kargoapi.PromotionRolloutPhaseSucceeded:
		fallback = kargoapi.PromotionPhaseSucceeded
	case kargoapi.PromotionRolloutPhaseFailed:
		fallback = kargoapi.PromotionPhaseFailed
	case kargoapi.PromotionRolloutPhaseErrored:
		fallback = kargoapi.PromotionPhaseErrored
	default:
		fallback = kargoapi.PromotionPhasePending
	}
	phases := make(map[string]kargoapi.PromotionPhase, len(rollout.Spec.Targets))
	for _, target := range rollout.Spec.Targets {
		phases[target.Name] = fallback
	}
	for _, target := range rollout.Status.Targets {
		if target.Phase != "" {
			phases[target.Name] = target.Phase
		}
	}
	return phases
}
