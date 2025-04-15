package promotion

import (
	"github.com/akuity/kargo/pkg/promotion"
)

// stepRunnerRegistry is a registry of StepRunners.
type stepRunnerRegistry map[string]promotion.StepRunner

// register adds a StepRunner to the stepRunnerRegistry.
func (s stepRunnerRegistry) register(runner promotion.StepRunner) {
	s[runner.Name()] = runner
}

// getStepRunner returns the StepRunner for the promotion step with the given
// name. If no StepRunner is registered with the given name, nil is returned
// instead.
func (s stepRunnerRegistry) getStepRunner(name string) promotion.StepRunner {
	return s[name]
}

// stepRunnerReg is a registry of StepRunners.
var stepRunnerReg = stepRunnerRegistry{}

// RegisterStepRunner adds a StepRunner to the package's internal registry.
func RegisterStepRunner(runner promotion.StepRunner) {
	stepRunnerReg.register(runner)
}

// GetStepRunner returns a StepRunner from the package's internal registry. If
// no StepRunner is registered with the given name, nil is returned instead.
func GetStepRunner(name string) promotion.StepRunner {
	return stepRunnerReg.getStepRunner(name)
}
