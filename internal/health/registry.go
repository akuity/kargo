package health

import "github.com/akuity/kargo/pkg/health"

// checkerRegistry is a registry of Checkers.
type checkerRegistry map[string]health.Checker

// register adds a Checker to the checkerRegistry.
func (c checkerRegistry) register(checker health.Checker) {
	c[checker.Name()] = checker
}

// getChecker returns the Checker for the health check with the given name. If
// no Checker is registered with the given name, nil is returned instead.
func (c checkerRegistry) getChecker(name string) health.Checker {
	return c[name]
}

// checkerReg is a registry of Checkers.
var checkerReg = checkerRegistry{}

// RegisterChecker adds a Checker to the package's internal registry.
func RegisterChecker(checker health.Checker) {
	checkerReg.register(checker)
}
