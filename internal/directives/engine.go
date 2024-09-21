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

// Directive is an interface for components that implement the logic for
// execution of the individual PromotionSteps of a user-defined promotion
// process. Implementations may optionally define corresponding health check
// procedures as well.
type Directive interface {
	// Name returns the name of the Directive.
	Name() string
	// RunPromotionStep executes an individual step of a user-defined promotion
	// process using the provided PromotionStepContext. Implementations may
	// indirectly modify that context through the returned PromotionStepResult to
	// allow subsequent promotion steps to access the results of its execution.
	RunPromotionStep(context.Context, *PromotionStepContext) (PromotionStepResult, error)
	// RunHealthCheckStep executes a health check using the provided
	// HealthCheckStepContext.
	RunHealthCheckStep(context.Context, *HealthCheckStepContext) HealthCheckStepResult
}
