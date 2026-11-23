package projectconfigs

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
)

// SpecValidator validates a ProjectConfigSpec beyond the built-in rules. It
// returns field errors with paths rooted at "spec". It is only invoked for a
// ProjectConfig that already passed every built-in rule, so it may rely on
// the constraints the CRD schema expressed (e.g. unique promotion window
// names and a valid window kind).
type SpecValidator func(
	ctx context.Context,
	project string,
	spec *kargoapi.ProjectConfigSpec,
) field.ErrorList

type (
	// SpecValidatorRegistration associates a name with a SpecValidator.
	SpecValidatorRegistration = component.NameBasedRegistration[
		SpecValidator,
		struct{},
	]

	// SpecValidatorRegistry is a name-based registry of SpecValidators that
	// can also list its registrations in the order they were registered.
	SpecValidatorRegistry interface {
		component.NameBasedRegistry[SpecValidator, struct{}]
		// List returns all registrations in the order they were registered.
		List() []SpecValidatorRegistration
	}
)

// DefaultSpecValidatorRegistry holds the SpecValidators that Validate runs
// after its built-in rules. Implementations register themselves in init().
var DefaultSpecValidatorRegistry = MustNewSpecValidatorRegistry()

// specValidatorRegistry decorates a name-based registry with the order of
// registration, which the underlying map-based registry does not keep.
type specValidatorRegistry struct {
	component.NameBasedRegistry[SpecValidator, struct{}]
	mu    sync.RWMutex
	names []string
}

// MustNewSpecValidatorRegistry returns a SpecValidatorRegistry holding the
// provided registrations in order. It panics if any registration is invalid
// or if two registrations share a name.
func MustNewSpecValidatorRegistry(
	registrations ...SpecValidatorRegistration,
) SpecValidatorRegistry {
	r := &specValidatorRegistry{
		NameBasedRegistry: component.MustNewNameBasedRegistry[SpecValidator, struct{}](
			&component.NameBasedRegistryOptions{AllowOverwriting: false},
		),
	}
	for _, reg := range registrations {
		r.MustRegister(reg)
	}
	return r
}

// Register adds a registration and records its position. Overwriting an
// existing registration is not permitted, so a name is recorded at most once.
func (r *specValidatorRegistry) Register(reg SpecValidatorRegistration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.NameBasedRegistry.Register(reg); err != nil {
		return err
	}
	r.names = append(r.names, reg.Name)
	return nil
}

// MustRegister adds a registration and panics on error.
func (r *specValidatorRegistry) MustRegister(reg SpecValidatorRegistration) {
	if err := r.Register(reg); err != nil {
		panic(err)
	}
}

// List implements SpecValidatorRegistry.
func (r *specValidatorRegistry) List() []SpecValidatorRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	regs := make([]SpecValidatorRegistration, 0, len(r.names))
	for _, name := range r.names {
		// Every recorded name was registered successfully and registrations
		// are never removed or overwritten, so Get cannot fail here.
		reg, _ := r.Get(name)
		regs = append(regs, reg)
	}
	return regs
}
