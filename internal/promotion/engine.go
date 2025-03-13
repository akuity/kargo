package promotion

import (
	"context"
)

// Engine is an interface for executing a sequence of promotion steps.
type Engine interface {
	// Promote executes the specified sequence of Steps and returns a Result
	// that aggregates the results of all steps.
	Promote(context.Context, Context, []Step) (Result, error)
}
