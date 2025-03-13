package health

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Checker is an interface for components that implement the logic for execution
// of a health check.
type Checker interface {
	// Name returns the name of the Checker.
	Name() string
	// Check executes a health check.
	Check(context.Context, Criteria) Result
}

// Criteria describes a request for the execution of a health check by a
// specific Checker.
type Criteria struct {
	// Project is the name of the Project that the health check requested by this
	// Criteria is associated with.
	Project string
	// Stage is the name of the Stage that the health check requested by this
	// Criteria is associated with.
	Stage string
	// Kind identifies a registered Checker that implements the logic
	// for the health check process.
	Kind string
	// Input is an opaque map of values to be passed to the Checker.
	Input Input
}

// Result represents the results of a health check executed by a Checker.
type Result struct {
	// Status is the high-level outcome of the HealthCheck executed by a
	// Checker.
	Status kargoapi.HealthState
	// Output is the opaque output of a HealthCheck executed by a
	// Checker. The Engine will aggregate this output and include it
	// in the final results of the health check, which will ultimately be included
	// in StageStatus.
	Output map[string]any
	// Issues is a list of issues that were encountered during the execution of
	// the HealthCheck by a Checker.
	Issues []string
}

// Input is an opaque map of values used as input to a Checker.
type Input map[string]any

// DeepCopy returns a deep copy of the input.
func (i Input) DeepCopy() Input {
	if i == nil {
		return nil
	}
	// TODO(hidde): we piggyback on the runtime package for now, as we expect the
	// input to originate from a Kubernetes API object. We should consider writing
	// our own implementation in the future.
	return runtime.DeepCopyJSON(i)
}

// ToJSON marshals the input to JSON.
func (i Input) ToJSON() []byte {
	if len(i) == 0 {
		return nil
	}
	b, _ := json.Marshal(i)
	return b
}

// InputToStruct converts an Input to a (typed) struct.
func InputToStruct[T any](input Input) (T, error) {
	var result T

	// Convert the map to JSON
	jsonData, err := json.Marshal(input)
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
