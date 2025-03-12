package directives

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// HealthChecker is an interface for components that implement the logic
// for execution of the individual HealthCheckSteps.
type HealthChecker interface {
	NamedRunner
	// CheckHealth executes a health check using the provided
	// HealthCheckStepContext.
	CheckHealth(context.Context, *HealthCheckStepContext) HealthCheckStepResult
}

// HealthCheckContext is the context of a health check process that is executed
// by the Engine.
type HealthCheckContext struct {
	// Project is the Project that the Stage is associated with.
	Project string
	// Stage is the Stage that the health check is targeting.
	Stage string
}

// HealthCheckStep describes a single step in a health check process.
// HealthCheckSteps are executed in sequence by the Engine, which delegates the
// execution of each step to a HealthChecker.
type HealthCheckStep struct {
	// Kind identifies a registered HealthChecker that implements the logic for
	// this step of the health check process.
	Kind string
	// Config is an opaque map of configuration values to be passed to the
	// HealthChecker executing this step.
	Config Config
}

// HealthCheckStepContext is a type that represents the context in which a
// single HealthCheckStep is executed by a HealthChecker.
type HealthCheckStepContext struct {
	// Config is the configuration of the step that is currently being
	// executed.
	Config Config
	// Project is the Project that the Stage is associated with.
	Project string
	// Stage is the Stage that the health check is targeting.
	Stage string
}

// HealthCheckStepResult represents the results of a single HealthCheckStep
// executed by a HealthChecker.
type HealthCheckStepResult struct {
	// Status is the high-level outcome of the HealthCheckStep executed by a
	// HealthChecker.
	Status kargoapi.HealthState
	// Output is the opaque output of a HealthCheckStepResult executed by a
	// HealthChecker. The Engine will aggregate this output and include it in the
	// final results of the health check, which will ultimately be included in
	// StageStatus.
	Output map[string]any
	// Issues is a list of issues that were encountered during the execution of
	// the HealthCheckStep by a HealthChecker.
	Issues []string
}
