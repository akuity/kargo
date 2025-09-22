package promotion

// stepRunnerRegistry is a map of StepRunnerRegistrations indexed by promotion
// step kind.
type stepRunnerRegistry map[string]StepRunnerRegistration

// register adds a StepRunnerRegistration to the stepRunnerRegistry.
func (s stepRunnerRegistry) register(
	stepKind string,
	registration StepRunnerRegistration,
) {
	if stepKind == "" {
		panic("step kind must be specified")
	}
	if registration.Factory == nil {
		panic("step registration must specify a factory function")
	}
	if registration.Metadata.DefaultErrorThreshold <= 0 {
		registration.Metadata.DefaultErrorThreshold = uint32(1)
	}
	s[stepKind] = registration
}

// getStepRunner returns the StepRunnerRegistration for the specified promotion
// step kind. If no such registration exists, nil is returned instead.
func (s stepRunnerRegistry) getStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	if registration, exists := s[stepKind]; exists {
		return &registration
	}
	return nil
}

// stepRunnerReg is this package's internal stepRunnerRegistry.
var stepRunnerReg = stepRunnerRegistry{}

// RegisterStepRunner adds a StepRunnerRegistration to the package's internal
// registry.
func RegisterStepRunner(
	stepKind string,
	registration StepRunnerRegistration,
) {
	stepRunnerReg.register(stepKind, registration)
}

// GetStepRunnerRegistration returns the StepRunnerRegistration for the
// specified promotion step kind from the package's internal registry. If no
// such registration exists, nil is returned instead.
func GetStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	return stepRunnerReg.getStepRunnerRegistration(stepKind)
}
