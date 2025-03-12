package directives

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/credentials"
)

// InitializeBuiltins registers all built-in step runners (Promoters and/or
// HealthCheckers) with the package's internal step runner registry.
func InitializeBuiltins(kargoClient, argocdClient client.Client, credsDB credentials.Database) {
	builtIns := []NamedRunner{
		newArgocdUpdater(argocdClient),
		newHelmChartUpdater(credsDB),
		newFileCopier(),
		newFileDeleter(),
		newGitCloner(credsDB),
		newGitCommitter(),
		newGitPROpener(credsDB),
		newGitPRWaiter(credsDB),
		newGitPusher(credsDB),
		newGitTreeClearer(),
		newHelmTemplateRunner(),
		newHTTPRequester(),
		newJSONParser(),
		newJSONUpdater(),
		newKustomizeBuilder(),
		newKustomizeImageSetter(kargoClient),
		newOutputComposer(),
		newYAMLParser(),
		newYAMLUpdater(),
	}
	for _, builtIn := range builtIns {
		Register(builtIn)
	}
}

// NamedRunner is an interface for runners that can self-report their name.
type NamedRunner interface {
	Name() string
}

// Register registers a NamedRunner with the package's internal step runner
// registry.
func Register(runner NamedRunner) {
	runnerReg.register(runner)
}

// runnerReg is a registry of Promoter and HealthChecker implementations.
var runnerReg = runnerRegistry{}

// runnerRegistry is a registry of named components that can presumably
// execute some sort of step, like a step of a promotion process or a health
// check process.
type runnerRegistry map[string]NamedRunner

// register adds a named component to the runnerRegistry.
func (r runnerRegistry) register(runner NamedRunner) {
	r[runner.Name()] = runner
}

// getPromoter returns the Promoter with the given name, if no runner is
// registered with the given name or the runner with the given name does not
// implement Promoter, nil is returned.
func (r runnerRegistry) getPromoter(name string) Promoter {
	runner, ok := r[name]
	if !ok {
		return nil
	}
	promoter, ok := runner.(Promoter)
	if !ok {
		return nil
	}
	return promoter
}

// getHealthChecker returns the HealthChecker with the given name, if no runner
// is registered with the given name or the runner with the given name does not
// implement HealthChecker, nil is returned.
func (r runnerRegistry) getHealthChecker(name string) HealthChecker {
	runner, ok := r[name]
	if !ok {
		return nil
	}
	healthChecker, ok := runner.(HealthChecker)
	if !ok {
		return nil
	}
	return healthChecker
}
