package directives

import "context"

// FakeEngine is a mock implementation of the Engine interface that can be used
// to facilitate unit testing.
type FakeEngine struct {
	ExecuteFn func(ctx context.Context, steps []Step) (Status, error)
}

func (e *FakeEngine) Execute(ctx context.Context, steps []Step) (Status, error) {
	if e.ExecuteFn == nil {
		return StatusSuccess, nil
	}
	return e.ExecuteFn(ctx, steps)
}
