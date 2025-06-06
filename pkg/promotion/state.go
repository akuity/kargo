package promotion

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

// State is a type that represents state shared by Steps in a user-defined
// promotion process. It is not safe for concurrent use at present, as we expect
// Steps to be executed sequentially.
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

// ToJSON marshals the State to JSON.
func (s State) ToJSON() []byte {
	if len(s) == 0 {
		return nil
	}
	b, _ := json.Marshal(s)
	return b
}
