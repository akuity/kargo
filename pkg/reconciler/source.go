package reconciler

import "context"

// Source is something that can be watched for: it pushes the requests a
// controller should reconcile onto the channel it is given.
//
// Implementations include a NATS topic in package nats, a Kubernetes kind in
// package kube, and whatever a store writes to list its own rows.
type Source[request any] interface {
	// Start begins pushing requests onto out. A source with initial state,
	// such as the objects an informer lists or the rows a store holds,
	// pushes all of it before returning, so that a controller's workers
	// begin with a complete picture. A source that cannot start, or cannot
	// load its initial state, returns an error and the controller does not
	// run. Pushing after Start stops when ctx is done. A source never closes
	// out; the controller owns it.
	//
	// Nothing reads out once ctx is done, so a source selects on ctx.Done()
	// when it sends, or a push after shutdown blocks forever.
	Start(ctx context.Context, out chan<- request) error
}
