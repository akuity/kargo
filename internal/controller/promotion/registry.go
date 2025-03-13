package promotion

// RegisterStepRunner adds a StepRunner to the package's internal registry.
func RegisterStepRunner(runner StepRunner) {
	stepRunnerReg.register(runner)
}

// stepRunnerReg is a registry of StepRunners.
var stepRunnerReg = stepRunnerRegistry{}

// stepRunnerRegistry is a registry of StepRunners.
type stepRunnerRegistry map[string]StepRunner

// register adds a StepRunner to the stepRunnerRegistry.
func (s stepRunnerRegistry) register(runner StepRunner) {
	s[runner.Name()] = runner
}

// getStepRunner returns the StepRunner for the promotion step with the given
// name. If no StepRunner is registered with the given name, nil is returned
// instead.
func (s stepRunnerRegistry) getStepRunner(name string) StepRunner {
	return s[name]
}
