package reconciler

import (
	"context"
	"time"
)

// Reconciler reconciles one resource. It is handed only the request, reads
// the resource fresh and acts on what it finds. Returning an error retries
// the request with exponential backoff.
//
// A Reconciler that runs in every replica must be idempotent; see the
// package documentation.
type Reconciler[request any] interface {
	Reconcile(ctx context.Context, req request) (Result, error)
}

// Result says what should happen after a successful Reconcile.
type Result struct {
	// RequeueAfter, when positive, reconciles the same request again after
	// that long.
	RequeueAfter time.Duration
}

// Func is a Reconciler implemented by a function.
type Func[request any] func(context.Context, request) (Result, error)

// Reconcile implements Reconciler.
func (f Func[request]) Reconcile(ctx context.Context, req request) (Result, error) {
	return f(ctx, req)
}
