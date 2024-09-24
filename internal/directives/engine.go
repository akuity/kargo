package directives

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Engine is an interface for executing user-defined promotion processes as well
// as corresponding health check processes.
type Engine interface {
	// Promote executes the provided list of PromotionSteps in sequence and
	// returns a PromotionResult that aggregates the results of all steps.
	Promote(context.Context, PromotionContext, []PromotionStep) (PromotionResult, error)
	// CheckHealth executes the provided list of HealthCheckSteps in sequence and
	// and returns a HealthCheckResult that aggregates the results of all steps.
	CheckHealth(context.Context, HealthCheckContext, []HealthCheckStep) kargoapi.Health
}
