package health

// RegisterChecker adds a Checker to the package's internal registry.
func RegisterChecker(checker Checker) {
	checkerReg.register(checker)
}

// checkerReg is a registry of Checkers.
var checkerReg = checkerRegistry{}

// checkerRegistry is a registry of Checkers.
type checkerRegistry map[string]Checker

// register adds a Checker to the checkerRegistry.
func (c checkerRegistry) register(checker Checker) {
	c[checker.Name()] = checker
}

// getChecker returns the Checker for the health check with the given name. If
// no Checker is registered with the given name, nil is returned instead.
func (c checkerRegistry) getChecker(name string) Checker {
	return c[name]
}
