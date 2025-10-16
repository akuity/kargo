package promotion

// StepRunnerRegistry is a map of StepRunnerRegistrations indexed by promotion
// step kind.
type StepRunnerRegistry map[string]StepRunnerRegistration

// Register adds a StepRunnerRegistration to the StepRunnerRegistry.
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

// GetStepRunnerRegistration returns the StepRunnerRegistration for the
// specified promotion step kind. If no such registration exists, nil is
// returned instead.
func (s StepRunnerRegistry) GetStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	if registration, exists := s[stepKind]; exists {
		return &registration
	}
	return nil
}

// stepRunnerRegistry is this package's internal StepRunnerRegistry.
var stepRunnerRegistry = StepRunnerRegistry{}

// RegisterStepRunner adds a StepRunnerRegistration to the package's internal
// registry.
func RegisterStepRunner(
	stepKind string,
	registration StepRunnerRegistration,
) {
	stepRunnerRegistry.Register(stepKind, registration)
}

// GetStepRunnerRegistration returns the StepRunnerRegistration for the
// specified promotion step kind from the package's internal registry. If no
// such registration exists, nil is returned instead.
func GetStepRunnerRegistration(
	stepKind string,
) *StepRunnerRegistration {
	return stepRunnerRegistry.GetStepRunnerRegistration(stepKind)
}

// GetStepRunnerRegistrations returns the package's internal StepRunnerRegistry.
func GetStepRunnerRegistrations() StepRunnerRegistry {
	return stepRunnerRegistry
}
