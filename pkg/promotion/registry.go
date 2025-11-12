package promotion

import (
	"time"

	"github.com/akuity/kargo/pkg/component"
)

type (
	StepRunnerFactory = func(StepRunnerCapabilities) StepRunner

	StepRunnerRegistration = component.NameBasedRegistration[
		StepRunnerFactory,
		StepRunnerMetadata,
	]

	StepRunnerRegistry = component.NameBasedRegistry[
		StepRunnerFactory,
		StepRunnerMetadata,
	]
)

type stepRunnerRegistry struct {
	StepRunnerRegistry
}

// Register decorates the internalRegistry's Register() method with metadata
// defaulting.
func (s *stepRunnerRegistry) Register(registration StepRunnerRegistration) error {
	if registration.Metadata.DefaultErrorThreshold == 0 {
		registration.Metadata.DefaultErrorThreshold = 1
	}
	return s.StepRunnerRegistry.Register(registration)
}

func (s *stepRunnerRegistry) MustRegister(registration StepRunnerRegistration) {
	if err := s.Register(registration); err != nil {
		panic(err)
	}
}

// MustNewStepRunnerRegistry overrides the internalRegistry's MustRegister()
// method to call the implementation-specific Register() method.
func MustNewStepRunnerRegistry(
	registrations ...StepRunnerRegistration,
) StepRunnerRegistry {
	r := &stepRunnerRegistry{
		StepRunnerRegistry: component.MustNewNameBasedRegistry(
			&component.NameBasedRegistryOptions{AllowOverwriting: true},
			registrations...,
		),
	}
	for _, reg := range registrations {
		r.MustRegister(reg)
	}
	return r
}

var DefaultStepRunnerRegistry = MustNewStepRunnerRegistry()

// StepRunnerMetadata contains metadata about a StepRunner.
type StepRunnerMetadata struct {
	// DefaultTimeout is the default soft maximum interval in which a StepRunner
	// that returns a Running status (which typically indicates it's waiting for
	// something to happen) may be retried.
	//
	// The maximum is a soft one because the check for whether the interval has
	// elapsed occurs AFTER the step has run. This effectively means a step may
	// run ONCE beyond the close of the interval.
	//
	// A value of 0 will cause the step to be retried indefinitely unless the
	// ErrorThreshold is reached.
	//
	// This default can be overridden by step-level configuration.
	DefaultTimeout time.Duration
	// DefaultErrorThreshold is the number of consecutive times the step must fail
	// (for any reason) before retries are abandoned and the entire Promotion is
	// marked as failed.
	//
	// If this field is set to a non-positive value, it will be changed to the
	// system-wide default of 1 at registration time.
	//
	// A value of 1 will cause the Promotion to be marked as failed after just
	// a single failure; i.e. no retries will be attempted.
	//
	// This default can be overridden by step-level configuration.
	DefaultErrorThreshold uint32
	// RequiredCapabilities is a list of constants representing special
	// capabilities required by the StepRunner in order to execute a step
	// successfully. The engine executing the StepRunner is responsible for
	// injecting necessary dependencies into the StepRunner when invoking its
	// factory function. By default, StepRunners are not granted any special
	// capabilities.
	RequiredCapabilities []StepRunnerCapability
}

// StepRunnerCapability is a type representing special capabilities that may be
// required by a StepRunner in order to execute a step successfully. The engine
// executing a StepRunner is responsible for injecting corresponding
// dependencies into it when invoking its factory function.
type StepRunnerCapability string

const (
	// StepCapabilityAccessArgoCD represents the capability of interacting with
	// an Argo CD control plane via a Kubernetes client.
	StepCapabilityAccessArgoCD StepRunnerCapability = "access-argocd"
	// StepCapabilityAccessControlPlane represents the capability of interacting
	// with the Kargo control plane via a Kubernetes client.
	StepCapabilityAccessControlPlane StepRunnerCapability = "access-control-plane"
	// StepCapabilityAccessCredentials represents the capability to obtain
	// repository credentials through a lookup by credential type and repository
	// URL.
	StepCapabilityAccessCredentials StepRunnerCapability = "access-credentials"
	// StepCapabilityTaskOutputPropagation represents the capability of a step,
	// when executed as part of a task, to propagate its output directly to the
	// Promotion's shared state, in addition to the task's own state.
	StepCapabilityTaskOutputPropagation StepRunnerCapability = "task-output-propagation"
)
