package directives

import "fmt"

// StepRegistry is a map of step names to steps. It is used to register and
// retrieve steps by name.
type StepRegistry map[string]Step

// RegisterStep registers a step with the given name. If a step with the same
// name has already been registered, it will be overwritten.
func (r StepRegistry) RegisterStep(step Step) {
	r[step.Name()] = step
}

// GetStep returns the step with the given name, or an error if no such step
// exists.
func (r StepRegistry) GetStep(name string) (Step, error) {
	step, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("step %q not found", name)
	}
	return step, nil
}
