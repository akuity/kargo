package directives

import (
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReservedStepAliasRegex is a regular expression that matches step aliases that
// are reserved for internal use.
var ReservedStepAliasRegex = regexp.MustCompile(`^(step|task)-\d+$`)

// SimpleEngine is a simple engine that executes a list of PromotionSteps in
// sequence.
type SimpleEngine struct {
	registry    runnerRegistry
	kargoClient client.Client
}

// NewSimpleEngine returns a new SimpleEngine that uses the package's built-in
// step runner registry.
func NewSimpleEngine(kargoClient client.Client) *SimpleEngine {
	return &SimpleEngine{
		registry:    runnerReg,
		kargoClient: kargoClient,
	}
}
