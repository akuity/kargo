package directives

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// PromotionStepRunner is an interface for components that implement the logic for
// execution of the individual PromotionSteps of a user-defined promotion
// process.
type PromotionStepRunner interface {
	// Name returns the name of the PromotionStepRunner.
	Name() string
	// RunPromotionStep executes an individual step of a user-defined promotion
	// process using the provided PromotionStepContext. Implementations may
	// indirectly modify that context through the returned PromotionStepResult to
	// allow subsequent promotion steps to access the results of its execution.
	RunPromotionStep(context.Context, *PromotionStepContext) (PromotionStepResult, error)
}

// PromotionContext is the context of a user-defined promotion process that is
// executed by the Engine.
type PromotionContext struct {
	// UIBaseURL may be used to construct deeper URLs for interacting with the
	// Kargo UI.
	UIBaseURL string
	// WorkDir is the working directory to use for the Promotion.
	WorkDir string
	// Project is the Project that the Promotion is associated with.
	Project string
	// Stage is the Stage that the Promotion is targeting.
	Stage string
	// Promotion is the name of the Promotion.
	Promotion string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to PromotionSteps that reference a particular artifact
	// and, in the absence of any explicit information about the origin of that
	// artifact, may need to examine FreightRequests to determine whether there
	// exists any ambiguity as to its origin, which a user may then need to
	// resolve.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	Freight kargoapi.FreightCollection
	// SharedState is the index of the step from which the promotion should begin
	// execution.
	StartFromStep int64
	// State is the current state of the promotion process.
	State State
}

// PromotionStep describes a single step in a user-defined promotion process.
// PromotionSteps are executed in sequence by the Engine, which delegates the
// execution of each step to a PromotionStepRunner.
type PromotionStep struct {
	// Kind identifies a registered PromotionStepRunner that implements the logic
	// for this step of the user-defined promotion process.
	Kind string
	// Alias is an optional identifier for this step of the use-defined promotion
	// process, which must be unique to the process. Output from execution of the
	// step will be keyed to this alias by the Engine and made accessible to
	// subsequent steps.
	Alias string
	// Config is an opaque map of configuration values to be passed to the
	// PromotionStepRunner executing this step.
	Config Config
}

// PromotionResult is the result of a user-defined promotion process executed by
// the Engine. It aggregates the status and output of the individual
// PromotionStepResults returned by the PromotionStepRunner executing each
// PromotionStep.
type PromotionResult struct {
	// Status is the high-level outcome of the user-defined promotion executed by
	// the Engine.
	Status kargoapi.PromotionPhase
	// Message is an optional message that provides additional context about the
	// outcome of the user-defined promotion executed by the Engine.
	Message string
	// HealthCheckSteps collects health check configuration returned from the
	// execution of individual PromotionSteps by their corresponding
	// PromotionStepRunners. This configuration can later be used as input to
	// health check processes.
	HealthCheckSteps []HealthCheckStep
	// If the promotion process remains in-progress, perhaps waiting for a change
	// in some external state, the value of this field will indicated where to
	// resume the process in the next reconciliation.
	CurrentStep int64
	// State is the current state of the promotion process.
	State State
}

// PromotionStepContext is a type that represents the context in which a
// SinglePromotion step is executed by a PromotionStepRunner.
type PromotionStepContext struct {
	// UIBaseURL may be used to construct deeper URLs for interacting with the
	// Kargo UI.
	UIBaseURL string
	// WorkDir is the root directory for the execution of a step.
	WorkDir string
	// SharedState is the state shared between steps.
	SharedState State
	// Alias is the alias of the step that is currently being executed.
	Alias string
	// Config is the configuration of the step that is currently being
	// executed.
	Config Config
	// Project is the Project that the Promotion is associated with.
	Project string
	// Stage is the Stage that the Promotion is targeting.
	Stage string
	// Promotion is the name of the Promotion.
	Promotion string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to PromotionStep that reference a particular artifact and,
	// in the absence of any explicit information about the origin of that
	// artifact, may need to examine FreightRequests to determine whether there
	// exists any ambiguity as to its origin, which a user may then need to
	// resolve.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express the artifacts they need to work with, we can make
	// the Engine responsible for finding them and furnishing them directly to
	// each PromotionStepRunner.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express the artifacts they need to work with, we can make
	// the Engine responsible for finding them and furnishing them directly to
	// each PromotionStepRunner.
	Freight kargoapi.FreightCollection
	// KargoClient is a Kubernetes client that a PromotionStepRunner executing a
	// PromotionStep may use to interact with the Kargo control plane. The value
	// of this field will often be nil, as the Engine will only furnish a this to
	// specially privileged PromotionStepRunners.
	//
	// TODO: krancour: Longer term, we may be able to do without this. See notes
	// on previous two fields.
	KargoClient client.Client
	// ArgoCDClient is a Kubernetes client that a PromotionStepRunner executing a
	// PromotionStep may use to interact with an Argo CD control plane. The value
	// of this field will often be nil, as the Engine will only furnish this to
	// specially privileged PromotionStepRunners.
	ArgoCDClient client.Client
	// CredentialsDB is a database of credentials that a PromotionStepRunner
	// executing a PromotionStep may use to acquire credentials for interacting
	// with external systems. The value of this field will often be nil, as the
	// Engine will only furnish a CredentialsDB to specially privileged
	// PromotionStepRunners.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express what credentials they need, we can make the Engine
	// responsible for finding them and furnishing them directly to each
	// PromotionStepRunner.
	CredentialsDB credentials.Database
}

// PromotionStepResult represents the results of single PromotionStep executed
// by a PromotionStepRunner.
type PromotionStepResult struct {
	// Status is the high-level outcome a PromotionStep executed by a
	// PromotionStepRunner.
	Status kargoapi.PromotionPhase
	// Message is an optional message that provides additional context about the
	// outcome of a PromotionStep executed by a PromotionStepRunner.
	Message string
	// Output is the opaque output of a PromotionStep executed by a
	// PromotionStepRunner. The Engine will update shared state with this output,
	// making it available to subsequent steps.
	Output map[string]any
	// HealthCheckStep is health check opaque configuration optionally returned by
	// a PromotionStepRunner's successful execution of a PromotionStep. This
	// configuration can later be used as input to health check processes.
	HealthCheckStep *HealthCheckStep
}
