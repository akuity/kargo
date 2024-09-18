package directives

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Engine is an interface for running a list of directives.
type Engine interface {
	// Execute runs the provided list of directives in sequence.
	Execute(ctx context.Context, promoCtx PromotionContext, steps []Step) (Status, error)
}

// PromotionContext is the context of the Promotion that is being executed by
// the Engine.
type PromotionContext struct {
	// WorkDir is the working directory to use for the Promotion.
	WorkDir string
	// Project is the Project that the Promotion is associated with.
	Project string
	// Stage is the Stage that the Promotion is targeting.
	Stage string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to Steps that reference a particular artifact and, in the
	// absence of any explicit information about the origin of that artifact, may
	// need to examine FreightRequests to determine whether there exists any
	// ambiguity as to its origin, which a user may then need to resolve.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	Freight kargoapi.FreightCollection
}

// Step is a single step that should be executed by the Engine.
type Step struct {
	// Directive is the name of the directive to execute for this step.
	Directive string
	// Alias is an optional alias for the step, which can be used to
	// refer to its results in subsequent steps.
	Alias string
	// Config is a map of configuration values that can be passed to the step.
	Config Config
}
