package promotion

// StepRunnerRegistry is a map of StepRunnerRegistrations indexed by promotion
// step kind.
type StepRunnerRegistry map[string]StepRunnerRegistration

// Register adds a StepRunnerRegistration to the stepRunnerRegistry.
func (s StepRunnerRegistry) Register(
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
func (s StepRunnerRegistry) GetStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	if registration, exists := s[stepKind]; exists {
		return &registration
	}
	return nil
}

// stepRunnerReg is this package's internal stepRunnerRegistry.
var stepRunnerReg = StepRunnerRegistry{}

// GetStepRunners returns the package's internal registry of StepRunners.
func GetStepRunners() map[string]StepRunnerRegistration {
	return stepRunnerReg
}

// RegisterStepRunner adds a StepRunnerRegistration to the package's internal
// registry.
func RegisterStepRunner(
	stepKind string,
	registration StepRunnerRegistration,
) {
	stepRunnerReg.Register(stepKind, registration)
}

// GetStepRunnerRegistration returns the StepRunnerRegistration for the
// specified promotion step kind from the package's internal registry. If no
// such registration exists, nil is returned instead.
func GetStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	return stepRunnerReg.GetStepRunnerRegistration(stepKind)
}
