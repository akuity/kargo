package directives

import (
	"fmt"
)

// builtins is a registry of built-in PromotionStepRunner and
// HealthCheckStepRunner implementations.
var builtins = NewStepRunnerRegistry()

// StepRunnerRegistry is a registry of built-in PromotionStepRunner and
// HealthCheckStepRunner implementations.
type StepRunnerRegistry struct {
	promotionStepRunners   map[string]PromotionStepRunnerRegistration
	healthCheckStepRunners map[string]HealthCheckStepRunnerRegistration
}

// NewStepRunnerRegistry returns a new StepRunnerRegistry.
func NewStepRunnerRegistry() *StepRunnerRegistry {
	return &StepRunnerRegistry{
		promotionStepRunners:   make(map[string]PromotionStepRunnerRegistration),
		healthCheckStepRunners: make(map[string]HealthCheckStepRunnerRegistration),
	}
}

// PromotionStepRunnerRegistration is a registration for a single
// PromotionStepRunner. It includes the PromotionStepRunner itself and a set of
// permissions that indicate capabilities the Engine should enable for the
// PromotionStepRunner.
type PromotionStepRunnerRegistration struct {
	// Permissions is a set of permissions that indicate capabilities the
	// Engine should enable for the PromotionStepRunner.
	Permissions StepRunnerPermissions
	// Runner is a PromotionStepRunner executes a discrete PromotionStep in the
	// context of a Promotion.
	Runner PromotionStepRunner
}

// HealthCheckStepRunnerRegistration is a registration for a single
// HealthCheckStepRunner. It includes the HealthCheckStepRunner itself and a set
// of permissions that indicate capabilities the Engine should enable for the
// HealthCheckStepRunner.
type HealthCheckStepRunnerRegistration struct {
	// Permissions is a set of permissions that indicate capabilities the Engine
	// should enable for the HealthCheckStepRunner.
	Permissions StepRunnerPermissions
	// Runner is a HealthCheckStepRunner executes a discrete HealthCheckStep.
	Runner HealthCheckStepRunner
}

// StepRunnerPermissions is a set of permissions that indicate capabilities the
// Engine should enable for a PromotionStepRunner or HealthCheckStepRunner.
type StepRunnerPermissions struct {
	// AllowCredentialsDB indicates whether the Engine may provide the step runner
	// with access to the credentials database.
	AllowCredentialsDB bool
	// AllowKargoClient indicates whether the Engine may provide the step runner
	// with access to a Kubernetes client for the Kargo control plane.
	AllowKargoClient bool
	// AllowArgoCDClient indicates whether the Engine may provide the step runner
	// with access to a Kubernetes client for the Argo CD control plane.
	AllowArgoCDClient bool
}

// RegisterPromotionStepRunner registers a PromotionStepRunner with the given
// name. If a PromotionStepRunner with the same name has already been
// registered, it will be overwritten.
func (s StepRunnerRegistry) RegisterPromotionStepRunner(
	runner PromotionStepRunner,
	permissions *StepRunnerPermissions,
) {
	if permissions == nil {
		permissions = &StepRunnerPermissions{}
	}
	s.promotionStepRunners[runner.Name()] = PromotionStepRunnerRegistration{
		Permissions: *permissions,
		Runner:      runner,
	}
}

// GetPromotionStepRunnerRegistration returns the
// PromotionStepRunnerRegistration for the promotion step with the given name,
// or an error if no such PromotionStepRunner is registered.
func (s StepRunnerRegistry) GetPromotionStepRunnerRegistration(
	name string,
) (PromotionStepRunnerRegistration, error) {
	reg, ok := s.promotionStepRunners[name]
	if !ok {
		return PromotionStepRunnerRegistration{},
			fmt.Errorf("runner not found for promotion step %q", name)
	}
	return reg, nil
}

// RegisterHealthCheckStepRunner registers a HealthCheckStepRunner with the
// given name. If a HealthCheckStepRunner with the same name has already been
// registered, it will be overwritten.
func (s StepRunnerRegistry) RegisterHealthCheckStepRunner(
	runner HealthCheckStepRunner,
	permissions *StepRunnerPermissions,
) {
	if permissions == nil {
		permissions = &StepRunnerPermissions{}
	}
	s.healthCheckStepRunners[runner.Name()] = HealthCheckStepRunnerRegistration{
		Permissions: *permissions,
		Runner:      runner,
	}
}

// GetHealthCheckStepRunnerRegistration returns the HealthStepRunnerRegistration
// for the health check step with the given name, or an error if no such
// HealthCheckStepRunner is registered.
func (s StepRunnerRegistry) GetHealthCheckStepRunnerRegistration(
	name string,
) (HealthCheckStepRunnerRegistration, error) {
	reg, ok := s.healthCheckStepRunners[name]
	if !ok {
		return HealthCheckStepRunnerRegistration{},
			fmt.Errorf("runner not found for health check step %q", name)
	}
	return reg, nil
}
