package promotion

import "time"

// StepRunnerRegistration associates a kind of promotion step with optional
// metadata and a factory function for instantiating a StepRunner capable of
// executing that kind of step.
type StepRunnerRegistration struct {
	// Metadata is optional metadata about StepRunners for the kind of step
	// specified by StepKind. If nil, default metadata will be applied during
	// registration.
	Metadata *StepRunnerMetadata
	// Factory is a function for instantiating a StepRunner capable of executing
	// the kind of step specified by StepKind.
	Factory func(StepRunnerCapabilities) StepRunner
}

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
	// If this field is set to nil, it will be changed to the system-wide default
	// of 0 at registration time.
	//
	// A value of 0 will cause the step to be retried indefinitely unless the
	// ErrorThreshold is reached.
	//
	// This default can be overridden by step-level configuration.
	DefaultTimeout *time.Duration
	// DefaultErrorThreshold is the number of consecutive times the step must fail
	// (for any reason) before retries are abandoned and the entire Promotion is
	// marked as failed.
	//
	// If this field is set to 0, it will be changed to the system-wide default of
	// 1 at registration time.
	//
	// A value of 1 will cause the Promotion to be marked as failed after just
	// a single failure; i.e. no retries will be attempted.
	//
	// There is no option to specify an infinite number of retries using a value
	// such as -1.
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
	// StepCapabilityAccessControlPlane represents the capability of interacting
	// with the Kargo control plane via a Kubernetes client.
	StepCapabilityAccessControlPlane StepRunnerCapability = "access-control-plane"
	// StepCapabilityAccessArgoCD represents the capability of interacting with
	// an Argo CD control plane via a Kubernetes client.
	StepCapabilityAccessArgoCD StepRunnerCapability = "access-argocd"
	// StepCapabilityAccessCredentials represents the capability to obtain
	// repository credentials through a lookup by credential type and repository
	// URL.
	StepCapabilityAccessCredentials StepRunnerCapability = "access-credentials"
)
