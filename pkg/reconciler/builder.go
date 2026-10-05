package reconciler

// Builder assembles a Controller fluently. Start one with New:
//
//	c, err := reconciler.New[database.TargetKey]("targets").
//		Watch(nats.Subject(conn, database.TargetEvents).Ignore(nats.Deleted)).
//		Watch(kube.Keyed(mgr.GetCache(), &kargoapi.Promotion{}, promotionTargets)).
//		Watch(list.New(store.ListTargetKeys).Every(2*time.Minute)).
//		Workers(4).
//		Func(reconcileTarget)
//	if err != nil { ... }
//	go c.Start(ctx)
type Builder[request comparable] struct {
	controller Controller[request]
}

// New starts building a controller with the given name.
func New[request comparable](name string) *Builder[request] {
	return &Builder[request]{
		controller: Controller[request]{Name: name},
	}
}

// Watch adds a source. Sources start in the order they are added; add
// event-driven sources before listing ones.
func (b *Builder[request]) Watch(src Source[request]) *Builder[request] {
	b.controller.Sources = append(b.controller.Sources, src)
	return b
}

// Workers sets how many requests are reconciled concurrently. The default
// is one.
func (b *Builder[request]) Workers(n int) *Builder[request] {
	b.controller.Workers = n
	return b
}

// build sets the reconciler and returns the controller, validated but not
// started.
func (b *Builder[request]) build(r Reconciler[request]) (*Controller[request], error) {
	c := b.controller
	c.Reconciler = r
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Func sets the reconcile function and returns the controller, validated
// but not started.
func (b *Builder[request]) Func(
	fn Func[request],
) (*Controller[request], error) {
	return b.build(fn)
}
