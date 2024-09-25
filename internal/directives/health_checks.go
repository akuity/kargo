package directives

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// HealthCheckStepRunner is an interface for components that implement the logic
// for execution of the individual HealthCheckSteps.
type HealthCheckStepRunner interface {
	// Name returns the name of the HealthCheckStepRunner.
	Name() string
	// RunHealthCheckStep executes a health check using the provided
	// HealthCheckStepContext.
	RunHealthCheckStep(context.Context, *HealthCheckStepContext) HealthCheckStepResult
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
// execution of each step to a HealthCheckStepRunner.
type HealthCheckStep struct {
	// Kind identifies a registered HealthCheckStepRunner that implements the
	// logic for this step of the health check process.
	Kind string
	// Config is an opaque map of configuration values to be passed to the
	// HealthCheckStepRunner executing this step.
	Config Config
}

// HealthCheckStepContext is a type that represents the context in which a
// single HealthCheckStep is executed by a HealthCheckStepRunner.
type HealthCheckStepContext struct {
	// Config is the configuration of the step that is currently being
	// executed.
	Config Config
	// Project is the Project that the Stage is associated with.
	Project string
	// Stage is the Stage that the health check is targeting.
	Stage string
	// KargoClient is a Kubernetes client that a HealthCheckStepRunner executing a
	// HealthCheckStep may use to interact with the Kargo control plane. The value
	// of this field will often be nil, as the Engine will only furnish a this to
	// specially privileged HealthCheckStepRunners.
	//
	// TODO: krancour: Longer term, we may be able to do without this.
	KargoClient client.Client
	// ArgoCDClient is a Kubernetes client that a HealthCheckStepRunner executing
	// a
	// HealthCheckStep may use to interact with an Argo CD control plane. The
	// value of this field will often be nil, as the Engine will only furnish this
	// to specially privileged HealthCheckStepRunners.
	ArgoCDClient client.Client
	// CredentialsDB is a database of credentials that a HealthCheckStepRunner
	// executing a HealthCheckStep may use to acquire credentials for interacting
	// with external systems. The value of this field will often be nil, as the
	// Engine will only furnish a CredentialsDB to specially privileged
	// HealthCheckStepRunners.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// HealthCheckSteps express what credentials they need, we can make the Engine
	// responsible for finding them and furnishing them directly to each
	// HealthCheckStepRunner.
	CredentialsDB credentials.Database
}

// HealthCheckStepResult represents the results of a single HealthCheckStep
// executed by a HealthCheckStepRunner.
type HealthCheckStepResult struct {
	// Status is the high-level outcome of the HealthCheckStep executed by a
	// HealthCheckStepRunner.
	Status kargoapi.HealthState
	// Output is the opaque output of a HealthCheckStepResult executed by a
	// HealthCheckStepRunner. The Engine will aggregate this output and include it
	// in the final results of the health check, which will ultimately be included
	// in StageStatus.
	Output State
	// Issues is a list of issues that were encountered during the execution of
	// the HealthCheckStep by a HealthCheckStepRunner.
	Issues []string
}
