package stages

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/logging"
)

// assessHealth assesses the health of a Stage based on the health checks from
// the last Promotion.
func (r *RegularStageReconciler) assessHealth(ctx context.Context, stage *kargoapi.Stage) kargoapi.StageStatus {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	if currentPromo := stage.Status.CurrentPromotion; currentPromo != nil {
		logger.Debug("Promotion is in progress: no health checks to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "ActivePromotion",
			Message:            "Stage has a Promotion in progress",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Cannot assess health because a Promotion is currently in progress",
			},
		}
		return newStatus
	}

	lastPromo := stage.Status.LastPromotion
	if lastPromo == nil {
		// A Stage that promotes through PromotionRequests never records a last
		// Promotion: its Promotions are children of a request, and health is a
		// property of each Target they promote to, not of the Stage. Say so,
		// rather than claiming the Stage has no Freight when its history may
		// well record some.
		if api.IsTargetAware(stage) {
			logger.Debug("Stage promotes to Targets: no Stage-level health checks to perform")
			conditions.Set(&newStatus, &metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionUnknown,
				Reason:             "TargetAwareStage",
				Message:            "Health is assessed per Target, not for the Stage",
				ObservedGeneration: stage.Generation,
			})
			newStatus.Health = nil
			return newStatus
		}
		logger.Debug("Stage has no current Freight: no health checks to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "NoFreight",
			Message:            "Stage has no current Freight",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = nil
		return newStatus
	}

	// If the last Promotion did not succeed, then we cannot perform any health
	// checks because they are only available after a successful Promotion.
	// Since we lack enough information to determine health, we mark it as Unknown.
	//
	// TODO(hidde): Long term, this should probably be changed to allow to
	//  continue to run health checks from the last successful Promotion,
	//  even if the current Promotion did not succeed (e.g. because it was
	//  aborted).
	if lastPromo.Status.Phase != kargoapi.PromotionPhaseSucceeded {
		logger.Debug("Last promotion did not succeed: defaulting Stage health to Unknown")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             fmt.Sprintf("LastPromotion%s", lastPromo.Status.Phase),
			Message:            "Cannot assess health because last Promotion did not succeed",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{"Cannot assess health because last Promotion did not succeed"},
		}
		return newStatus
	}

	// Compose the health check criteria.
	healthChecks := lastPromo.Status.HealthChecks
	var criteria []health.Criteria
	for _, check := range healthChecks {
		criteria = append(criteria, health.Criteria{
			Kind:  check.Uses,
			Input: check.GetConfig(),
		})
	}

	// Run the hlth checks.
	hlth := r.healthChecker.Check(ctx, stage.Namespace, stage.Name, criteria)
	newStatus.Health = &hlth

	// Set the Healthy condition based on the health status.
	switch hlth.Status {
	case kargoapi.HealthStateHealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionTrue,
			Reason:             string(hlth.Status),
			Message:            fmt.Sprintf("Stage is healthy (performed %d health checks)", len(healthChecks)),
			ObservedGeneration: stage.Generation,
		})
	case kargoapi.HealthStateUnhealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:   kargoapi.ConditionTypeHealthy,
			Status: metav1.ConditionFalse,
			Reason: string(hlth.Status),
			Message: fmt.Sprintf(
				"Stage is unhealthy (%d issues in %d health checks)",
				len(hlth.Issues), len(healthChecks),
			),
			ObservedGeneration: stage.Generation,
		})
	default:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             string(hlth.Status),
			ObservedGeneration: stage.Generation,
		})
	}

	return newStatus
}