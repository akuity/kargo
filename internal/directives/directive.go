package directives

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
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

// Result is a type that represents the result of a Directive.
type Result string

const (
	// ResultSuccess is the result of a successful directive.
	ResultSuccess Result = "Success"
	// ResultFailure is the result of a failed directive.
	ResultFailure Result = "Failure"
)

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
