package directives

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// StepContext is a type that represents the context in which a step is
// executed.
type StepContext struct {
	// WorkDir is the root directory for the execution of a step.
	WorkDir string
	// SharedState is the state shared between steps.
	SharedState State
	// Alias is the alias of the step that is currently being executed.
	Alias string
	// Config is the configuration of the step that is currently being
	// executed.
	Config Config
	// Project is the Project that the Promotion is associated with.
	Project string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to Steps that reference a particular artifact and, in the
	// absence of any explicit information about the origin of that artifact, may
	// need to examine FreightRequests to determine whether there exists any
	// ambiguity as to its origin, which a user may then need to resolve.
	//
	// TODO: krancour: Longer term, if we can standardize the way that all steps
	// express the artifacts they need to work with, we can make the Step
	// execution engine responsible for finding them and furnishing them directly
	// to each Step.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	//
	// TODO: krancour: Longer term, if we can standardize the way that all steps
	// express the artifacts they need to work with, we can make the Step
	// execution engine responsible for finding them and furnishing them directly
	// to each Step.
	Freight kargoapi.FreightCollection
	// KargoClient is a Kubernetes client that Steps involved in the Promotion may
	// use to interact with the Kargo control plane. The value of this field will
	// often be nil, as the step execution engine will only furnish a this to
	// specially privileged Steps.
	//
	// TODO: krancour: Longer term, we may be able to do without this. See notes
	// on previous two fields.
	KargoClient client.Client
	// ArgoCDClient is a Kubernetes client that Steps involved in the Promotion
	// may use to interact with an Argo CD control plane. The value of this field
	// will often be nil, as the step execution engine will only furnish this to
	// specially privileged Steps.
	//
	// TODO: krancour: Longer term, we may be able to do without this. See notes
	// on previous two fields.
	ArgoCDClient client.Client
	// CredentialsDB is a database of credentials that Steps may use to acquire
	// credentials for interacting with external systems. The value of this field
	// will often be nil, as the step execution engine will only furnish a
	// CredentialsDB to specially privileged Steps.
	//
	// TODO: krancour: Longer term, if we can standardize the way that all steps
	// express what credentials they need, we can make the Step execution engine
	// responsible for finding them and furnishing them directly to each Step.
	CredentialsDB credentials.Database
}

// State is a type that represents shared state between steps.
// It is not safe for concurrent use at present, as we expect steps to
// be executed sequentially.
type State map[string]any

// Set stores a value in the shared state.
func (s State) Set(key string, value any) {
	s[key] = value
}

// Get retrieves a value from the shared state.
func (s State) Get(key string) (any, bool) {
	value, ok := s[key]
	return value, ok
}

// DeepCopy returns a deep copy of the state.
func (s *State) DeepCopy() State {
	if s == nil {
		return nil
	}
	// TODO(hidde): we piggyback on the runtime package for now, as we expect
	// the configuration to originate from a Kubernetes API object. We should
	// consider writing our own implementation in the future.
	return runtime.DeepCopyJSON(*s)
}

// Config is a map of configuration values that can be passed to a step.
// The keys and values are arbitrary, and the step is responsible for
// interpreting them.
type Config map[string]any

// DeepCopy returns a deep copy of the configuration.
func (c Config) DeepCopy() Config {
	if c == nil {
		return nil
	}
	// TODO(hidde): we piggyback on the runtime package for now, as we expect
	// the configuration to originate from a Kubernetes API object. We should
	// consider writing our own implementation in the future.
	return runtime.DeepCopyJSON(c)
}

// Status is a type that represents the high-level outcome of a directive
// execution.
type Status string

const (
	// StatusSuccess is the result of a successful directive execution.
	StatusSuccess Status = "Success"
	// StatusFailure is the result of a failed directive execution.
	StatusFailure Status = "Failure"
)

// Result represents the outcome of a directive execution, including its status
// (e.g. Success or Failure) and any output (State) that the execution engine
// executing the directive must append to the shared state.
type Result struct {
	// Status is the high-level outcome of the directive execution.
	Status Status
	// Output is the output of the directive execution.
	Output State
}

// Directive is an interface that a directive must implement. A directive is
// a responsible for executing a specific action, and may modify the provided
// context to allow subsequent directives to access the results of its
// execution.
type Directive interface {
	// Name returns the name of the directive.
	Name() string
	// Run executes the directive using the provided context and configuration.
	Run(ctx context.Context, stepCtx *StepContext) (Result, error)
}

// configToStruct converts a Config to a (typed) configuration struct.
func configToStruct[T any](c Config) (T, error) {
	var result T

	// Convert the map to JSON
	jsonData, err := json.Marshal(c)
	if err != nil {
		return result, err
	}

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
